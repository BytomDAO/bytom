package wallet

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/account"
	"github.com/bytom/bytom/asset"
	"github.com/bytom/bytom/blockchain/query"
	"github.com/bytom/bytom/crypto/sha3pool"
	dbm "github.com/bytom/bytom/database/leveldb"
	chainjson "github.com/bytom/bytom/encoding/json"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

const (
	//TxPrefix is wallet database transactions prefix
	TxPrefix = "TXS:"
	//TxIndexPrefix is wallet database tx index prefix
	TxIndexPrefix = "TID:"
	//TxIndexPrefix is wallet database global tx index prefix
	GlobalTxIndexPrefix = "GTID:"
)

var errAccntTxIDNotFound = errors.New("account TXID not found")

func formatKey(blockHeight uint64, position uint32) string {
	return fmt.Sprintf("%016x%08x", blockHeight, position)
}

func calcAnnotatedKey(formatKey string) []byte {
	return []byte(TxPrefix + formatKey)
}

func calcDeleteKey(blockHeight uint64) []byte {
	return []byte(fmt.Sprintf("%s%016x", TxPrefix, blockHeight))
}

func calcTxIndexKey(txID string) []byte {
	return []byte(TxIndexPrefix + txID)
}

func calcGlobalTxIndexKey(txID string) []byte {
	return []byte(GlobalTxIndexPrefix + txID)
}

func calcGlobalTxIndex(blockHash *bc.Hash, position uint64) []byte {
	txIdx := make([]byte, 40)
	copy(txIdx[:32], blockHash.Bytes())
	binary.BigEndian.PutUint64(txIdx[32:], position)
	return txIdx
}

func parseGlobalTxIdx(globalTxIdx []byte) (*bc.Hash, uint64) {
	var hashBytes [32]byte
	copy(hashBytes[:], globalTxIdx[:32])
	hash := bc.NewHash(hashBytes)
	position := binary.BigEndian.Uint64(globalTxIdx[32:])
	return &hash, position
}

// deleteTransaction delete transactions when orphan block rollback
func (w *Wallet) deleteTransactions(batch dbm.Batch, height uint64) {
	tmpTx := query.AnnotatedTx{}
	txIter := w.DB.IteratorPrefix(calcDeleteKey(height))
	defer txIter.Release()

	for txIter.Next() {
		if err := json.Unmarshal(txIter.Value(), &tmpTx); err == nil {
			batch.Delete(calcTxIndexKey(tmpTx.ID.String()))
		}
		batch.Delete(txIter.Key())
	}
}

// saveExternalAssetDefinition save external and local assets definition,
// when query ,query local first and if have no then query external
// details see getAliasDefinition
func saveExternalAssetDefinition(b *types.Block, walletDB dbm.DB) {
	storeBatch := walletDB.NewBatch()
	defer storeBatch.Write()

	for _, tx := range b.Transactions {
		for _, orig := range tx.Inputs {
			if ii, ok := orig.TypedInput.(*types.IssuanceInput); ok {
				if isValidJSON(ii.AssetDefinition) {
					assetID := ii.AssetID()
					if assetExist := walletDB.Get(asset.ExtAssetKey(&assetID)); assetExist == nil {
						storeBatch.Set(asset.ExtAssetKey(&assetID), ii.AssetDefinition)
					}
				}
			}
		}
	}
}

// Summary is the struct of transaction's input and output summary
type Summary struct {
	Type         string             `json:"type"`
	AssetID      bc.AssetID         `json:"asset_id,omitempty"`
	AssetAlias   string             `json:"asset_alias,omitempty"`
	Amount       uint64             `json:"amount,omitempty"`
	AccountID    string             `json:"account_id,omitempty"`
	AccountAlias string             `json:"account_alias,omitempty"`
	Arbitrary    chainjson.HexBytes `json:"arbitrary,omitempty"`
}

// TxSummary is the struct of transaction summary
type TxSummary struct {
	ID        bc.Hash   `json:"tx_id"`
	Timestamp uint64    `json:"block_time"`
	Inputs    []Summary `json:"inputs"`
	Outputs   []Summary `json:"outputs"`
}

// indexTransactions saves all annotated transactions to the database.
func (w *Wallet) indexTransactions(batch dbm.Batch, b *types.Block, txStatus *bc.TransactionStatus) error {
	annotatedTxs := w.filterAccountTxs(b, txStatus)
	saveExternalAssetDefinition(b, w.DB)
	annotateTxsAccount(annotatedTxs, w.DB)

	for _, tx := range annotatedTxs {
		rawTx, err := json.Marshal(tx)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("inserting annotated_txs to db")
			return err
		}

		batch.Set(calcAnnotatedKey(formatKey(b.Height, uint32(tx.Position))), rawTx)
		batch.Set(calcTxIndexKey(tx.ID.String()), []byte(formatKey(b.Height, uint32(tx.Position))))

		// delete unconfirmed transaction
		batch.Delete(calcUnconfirmedTxKey(tx.ID.String()))
	}

	if !w.TxIndexFlag {
		return nil
	}

	for position, globalTx := range b.Transactions {
		blockHash := b.BlockHeader.Hash()
		batch.Set(calcGlobalTxIndexKey(globalTx.ID.String()), calcGlobalTxIndex(&blockHash, uint64(position)))
	}

	return nil
}

// filterAccountTxs related and build the fully annotated transactions.
func (w *Wallet) filterAccountTxs(b *types.Block, txStatus *bc.TransactionStatus) []*query.AnnotatedTx {
	annotatedTxs := make([]*query.AnnotatedTx, 0, len(b.Transactions))

transactionLoop:
	for pos, tx := range b.Transactions {
		statusFail, _ := txStatus.GetStatus(pos)
		for _, v := range tx.Outputs {
			var hash [32]byte
			sha3pool.Sum256(hash[:], v.ControlProgram)

			if bytes := w.DB.Get(account.ContractKey(hash)); bytes != nil {
				annotatedTxs = append(annotatedTxs, w.buildAnnotatedTransaction(tx, b, statusFail, pos))
				continue transactionLoop
			}
		}

		for _, v := range tx.Inputs {
			outid, err := v.SpentOutputID()
			if err != nil {
				continue
			}
			if bytes := w.DB.Get(account.StandardUTXOKey(outid)); bytes != nil {
				annotatedTxs = append(annotatedTxs, w.buildAnnotatedTransaction(tx, b, statusFail, pos))
				continue transactionLoop
			}
		}
	}

	return annotatedTxs
}

// GetTransactionByTxID get transaction by txID
func (w *Wallet) GetTransactionByTxID(txID string) (*query.AnnotatedTx, error) {
	if annotatedTx, err := w.getAccountTxByTxID(txID); err == nil {
		return annotatedTx, nil
	} else if !w.TxIndexFlag {
		return nil, err
	}

	return w.getGlobalTxByTxID(txID)
}

func (w *Wallet) getAccountTxByTxID(txID string) (*query.AnnotatedTx, error) {
	annotatedTx := &query.AnnotatedTx{}
	formatKey := w.DB.Get(calcTxIndexKey(txID))
	if formatKey == nil {
		return nil, errAccntTxIDNotFound
	}

	txInfo := w.DB.Get(calcAnnotatedKey(string(formatKey)))
	if err := json.Unmarshal(txInfo, annotatedTx); err != nil {
		return nil, err
	}

	annotateTxsAsset(w, []*query.AnnotatedTx{annotatedTx})
	return annotatedTx, nil
}

func (w *Wallet) getGlobalTxByTxID(txID string) (*query.AnnotatedTx, error) {
	globalTxIdx := w.DB.Get(calcGlobalTxIndexKey(txID))
	if globalTxIdx == nil {
		return nil, fmt.Errorf("No transaction(tx_id=%s) ", txID)
	}

	blockHash, pos := parseGlobalTxIdx(globalTxIdx)
	block, err := w.chain.GetBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}

	txStatus, err := w.chain.GetTransactionStatus(blockHash)
	if err != nil {
		return nil, err
	}

	statusFail, err := txStatus.GetStatus(int(pos))
	if err != nil {
		return nil, err
	}

	tx := block.Transactions[int(pos)]
	return w.buildAnnotatedTransaction(tx, block, statusFail, int(pos)), nil
}

// GetTransactionsSummary get transactions summary
func (w *Wallet) GetTransactionsSummary(transactions []*query.AnnotatedTx) []TxSummary {
	Txs := []TxSummary{}

	for _, annotatedTx := range transactions {
		tmpTxSummary := TxSummary{
			Inputs:    make([]Summary, len(annotatedTx.Inputs)),
			Outputs:   make([]Summary, len(annotatedTx.Outputs)),
			ID:        annotatedTx.ID,
			Timestamp: annotatedTx.Timestamp,
		}

		for i, input := range annotatedTx.Inputs {
			tmpTxSummary.Inputs[i].Type = input.Type
			tmpTxSummary.Inputs[i].AccountID = input.AccountID
			tmpTxSummary.Inputs[i].AccountAlias = input.AccountAlias
			tmpTxSummary.Inputs[i].AssetID = input.AssetID
			tmpTxSummary.Inputs[i].AssetAlias = input.AssetAlias
			tmpTxSummary.Inputs[i].Amount = input.Amount
			tmpTxSummary.Inputs[i].Arbitrary = input.Arbitrary
		}
		for j, output := range annotatedTx.Outputs {
			tmpTxSummary.Outputs[j].Type = output.Type
			tmpTxSummary.Outputs[j].AccountID = output.AccountID
			tmpTxSummary.Outputs[j].AccountAlias = output.AccountAlias
			tmpTxSummary.Outputs[j].AssetID = output.AssetID
			tmpTxSummary.Outputs[j].AssetAlias = output.AssetAlias
			tmpTxSummary.Outputs[j].Amount = output.Amount
		}

		Txs = append(Txs, tmpTxSummary)
	}

	return Txs
}

func findTransactionsByAccount(annotatedTx *query.AnnotatedTx, accountID string) bool {
	for _, input := range annotatedTx.Inputs {
		if input.AccountID == accountID {
			return true
		}
	}

	for _, output := range annotatedTx.Outputs {
		if output.AccountID == accountID {
			return true
		}
	}

	return false
}

// GetTransactions get all walletDB transactions, and filter transactions by accountID optional
func (w *Wallet) GetTransactions(accountID string) ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}

	txIter := w.DB.IteratorPrefix([]byte(TxPrefix))
	defer txIter.Release()
	for txIter.Next() {
		annotatedTx := &query.AnnotatedTx{}
		if err := json.Unmarshal(txIter.Value(), &annotatedTx); err != nil {
			return nil, err
		}

		if accountID == "" || findTransactionsByAccount(annotatedTx, accountID) {
			annotateTxsAsset(w, []*query.AnnotatedTx{annotatedTx})
			annotatedTxs = append([]*query.AnnotatedTx{annotatedTx}, annotatedTxs...)
		}
	}

	return annotatedTxs, nil
}

// GetAccountBalances return all account balances
func (w *Wallet) GetAccountBalances(accountID string, id string) ([]AccountBalance, error) {
	return w.indexBalances(w.GetAccountUtxos(accountID, "", false, false))
}

// AccountBalance account balance
type AccountBalance struct {
	AccountID       string                 `json:"account_id"`
	Alias           string                 `json:"account_alias"`
	AssetAlias      string                 `json:"asset_alias"`
	AssetID         string                 `json:"asset_id"`
	Amount          uint64                 `json:"amount"`
	AssetDefinition map[string]interface{} `json:"asset_definition"`
}

func (w *Wallet) indexBalances(accountUTXOs []*account.UTXO) ([]AccountBalance, error) {
	accBalance := make(map[string]map[string]uint64)
	balances := []AccountBalance{}

	for _, accountUTXO := range accountUTXOs {
		assetID := accountUTXO.AssetID.String()
		if _, ok := accBalance[accountUTXO.AccountID]; ok {
			if _, ok := accBalance[accountUTXO.AccountID][assetID]; ok {
				accBalance[accountUTXO.AccountID][assetID] += accountUTXO.Amount
			} else {
				accBalance[accountUTXO.AccountID][assetID] = accountUTXO.Amount
			}
		} else {
			accBalance[accountUTXO.AccountID] = map[string]uint64{assetID: accountUTXO.Amount}
		}
	}

	var sortedAccount []string
	for k := range accBalance {
		sortedAccount = append(sortedAccount, k)
	}
	sort.Strings(sortedAccount)

	for _, id := range sortedAccount {
		var sortedAsset []string
		for k := range accBalance[id] {
			sortedAsset = append(sortedAsset, k)
		}
		sort.Strings(sortedAsset)

		for _, assetID := range sortedAsset {
			alias := w.AccountMgr.GetAliasByID(id)
			targetAsset, err := w.AssetReg.GetAsset(assetID)
			if err != nil {
				return nil, err
			}

			assetAlias := *targetAsset.Alias
			balances = append(balances, AccountBalance{
				Alias:           alias,
				AccountID:       id,
				AssetID:         assetID,
				AssetAlias:      assetAlias,
				Amount:          accBalance[id][assetID],
				AssetDefinition: targetAsset.DefinitionMap,
			})
		}
	}

	return balances, nil
}
