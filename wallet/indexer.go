package wallet

import (
	"encoding/json"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/consensus"
	"github.com/bytom/consensus/segwit"
	"github.com/bytom/crypto/sha3pool"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

type rawOutput struct {
	OutputID bc.Hash
	bc.AssetAmount
	ControlProgram []byte
	txHash         bc.Hash
	outputIndex    uint32
	sourceID       bc.Hash
	sourcePos      uint64
	ValidHeight    uint64
}

type accountOutput struct {
	rawOutput
	AccountID string
	Address   string
	keyIndex  uint64
}

const (
	//TxPrefix is wallet database transactions prefix
	TxPrefix = "TXS:"
	//TxIndexPrefix is wallet database tx index prefix
	TxIndexPrefix = "TID:"
)

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

// deleteTransaction delete transactions when orphan block rollback
func (w *Wallet) deleteTransactions(batch db.Batch, height uint64) {
	tmpTx := query.AnnotatedTx{}

	txIter := w.DB.IteratorPrefix(calcDeleteKey(height))
	defer txIter.Release()

	for txIter.Next() {
		if err := json.Unmarshal(txIter.Value(), &tmpTx); err == nil {
			// delete index
			batch.Delete(calcTxIndexKey(tmpTx.ID.String()))
		}

		batch.Delete(txIter.Key())
	}
}

// ReverseAccountUTXOs process the invalid blocks when orphan block rollback
func (w *Wallet) reverseAccountUTXOs(batch db.Batch, b *types.Block, txStatus *bc.TransactionStatus) {
	var err error

	// unknow how many spent and retire outputs
	reverseOuts := []*rawOutput{}

	// handle spent UTXOs
	for txIndex, tx := range b.Transactions {
		for _, inpID := range tx.Tx.InputIDs {
			// spend and retire
			sp, err := tx.Spend(inpID)
			if err != nil {
				continue
			}

			resOut, ok := tx.Entries[*sp.SpentOutputId].(*bc.Output)
			if !ok {
				continue
			}

			statusFail, _ := txStatus.GetStatus(txIndex)
			if statusFail && *resOut.Source.Value.AssetId != *consensus.BTMAssetID {
				continue
			}

			out := &rawOutput{
				OutputID:       *sp.SpentOutputId,
				AssetAmount:    *resOut.Source.Value,
				ControlProgram: resOut.ControlProgram.Code,
				txHash:         tx.ID,
				sourceID:       *resOut.Source.Ref,
				sourcePos:      resOut.Source.Position,
			}
			reverseOuts = append(reverseOuts, out)
		}
	}

	accOuts := loadAccountInfo(reverseOuts, w)
	if err = upsertConfirmedAccountOutputs(accOuts, batch); err != nil {
		log.WithField("err", err).Error("reversing account spent and retire outputs")
		return
	}

	// handle new UTXOs
	for _, tx := range b.Transactions {
		for j := range tx.Outputs {
			resOutID := tx.ResultIds[j]
			resOut, ok := tx.Entries[*resOutID].(*bc.Output)
			if !ok {
				// retirement
				continue
			}

			if segwit.IsP2WScript(resOut.ControlProgram.Code) {
				// delete standard UTXOs
				batch.Delete(account.StandardUTXOKey(*resOutID))
			} else {
				// delete contract UTXOs
				batch.Delete(account.ContractUTXOKey(*resOutID))
			}
		}
	}
}

// saveExternalAssetDefinition save external and local assets definition,
// when query ,query local first and if have no then query external
// details see getAliasDefinition
func saveExternalAssetDefinition(b *types.Block, walletDB db.DB) {
	storeBatch := walletDB.NewBatch()
	defer storeBatch.Write()

	for _, tx := range b.Transactions {
		for _, orig := range tx.Inputs {
			if ii, ok := orig.TypedInput.(*types.IssuanceInput); ok {
				if isValidJSON(ii.AssetDefinition) {
					assetID := ii.AssetID()
					if assetExist := walletDB.Get(asset.CalcExtAssetKey(&assetID)); assetExist == nil {
						storeBatch.Set(asset.CalcExtAssetKey(&assetID), ii.AssetDefinition)
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
func (w *Wallet) indexTransactions(batch db.Batch, b *types.Block, txStatus *bc.TransactionStatus) error {
	annotatedTxs := w.filterAccountTxs(b, txStatus)
	saveExternalAssetDefinition(b, w.DB)
	annotateTxsAsset(w, annotatedTxs)
	annotateTxsAccount(annotatedTxs, w.DB)

	for _, tx := range annotatedTxs {
		rawTx, err := json.Marshal(tx)
		if err != nil {
			log.WithField("err", err).Error("inserting annotated_txs to db")
			return err
		}

		batch.Set(calcAnnotatedKey(formatKey(b.Height, uint32(tx.Position))), rawTx)
		batch.Set(calcTxIndexKey(tx.ID.String()), []byte(formatKey(b.Height, uint32(tx.Position))))
	}
	return nil
}

// buildAccountUTXOs process valid blocks to build account unspent outputs db
func (w *Wallet) buildAccountUTXOs(batch db.Batch, b *types.Block, txStatus *bc.TransactionStatus) {
	// get the spent UTXOs and delete the UTXOs from DB
	prevoutDBKeys(batch, b, txStatus)

	// handle new UTXOs
	var outs []*rawOutput
	for txIndex, tx := range b.Transactions {
		for i, out := range tx.Outputs {
			resOutID := tx.ResultIds[i]
			resOut, ok := tx.Entries[*resOutID].(*bc.Output)
			if !ok {
				continue
			}

			if statusFail, _ := txStatus.GetStatus(txIndex); statusFail && *resOut.Source.Value.AssetId != *consensus.BTMAssetID {
				continue
			}

			out := &rawOutput{
				OutputID:       *tx.OutputID(i),
				AssetAmount:    out.AssetAmount,
				ControlProgram: out.ControlProgram,
				txHash:         tx.ID,
				outputIndex:    uint32(i),
				sourceID:       *resOut.Source.Ref,
				sourcePos:      resOut.Source.Position,
			}

			// coinbase utxo valid height
			if txIndex == 0 {
				out.ValidHeight = b.Height + consensus.CoinbasePendingBlockNumber
			}
			outs = append(outs, out)
		}
	}
	accOuts := loadAccountInfo(outs, w)

	if err := upsertConfirmedAccountOutputs(accOuts, batch); err != nil {
		log.WithField("err", err).Error("building new account outputs")
	}
}

func prevoutDBKeys(batch db.Batch, b *types.Block, txStatus *bc.TransactionStatus) {
	for txIndex, tx := range b.Transactions {
		for _, inpID := range tx.Tx.InputIDs {
			sp, err := tx.Spend(inpID)
			if err != nil {
				continue
			}

			statusFail, _ := txStatus.GetStatus(txIndex)
			if statusFail && *sp.WitnessDestination.Value.AssetId != *consensus.BTMAssetID {
				continue
			}

			resOut, ok := tx.Entries[*sp.SpentOutputId].(*bc.Output)
			if !ok {
				// retirement
				log.WithField("SpentOutputId", *sp.SpentOutputId).Info("the OutputId is retirement")
				continue
			}

			if segwit.IsP2WScript(resOut.ControlProgram.Code) {
				// delete standard UTXOs
				batch.Delete(account.StandardUTXOKey(*sp.SpentOutputId))
			} else {
				// delete contract UTXOs
				batch.Delete(account.ContractUTXOKey(*sp.SpentOutputId))
			}
		}
	}
	return
}

// loadAccountInfo turns a set of output IDs into a set of
// outputs by adding account annotations.  Outputs that can't be
// annotated are excluded from the result.
func loadAccountInfo(outs []*rawOutput, w *Wallet) []*accountOutput {
	outsByScript := make(map[string][]*rawOutput, len(outs))
	for _, out := range outs {
		scriptStr := string(out.ControlProgram)
		outsByScript[scriptStr] = append(outsByScript[scriptStr], out)
	}

	result := make([]*accountOutput, 0, len(outs))
	cp := account.CtrlProgram{}

	var hash [32]byte
	for s := range outsByScript {
		// smart contract UTXO
		if !segwit.IsP2WScript([]byte(s)) {
			for _, out := range outsByScript[s] {
				newOut := &accountOutput{
					rawOutput: *out,
				}
				result = append(result, newOut)
			}

			continue
		}

		sha3pool.Sum256(hash[:], []byte(s))
		bytes := w.DB.Get(account.CPKey(hash))
		if bytes == nil {
			continue
		}

		err := json.Unmarshal(bytes, &cp)
		if err != nil {
			continue
		}

		isExist := w.DB.Get(account.Key(cp.AccountID))
		if isExist == nil {
			continue
		}

		for _, out := range outsByScript[s] {
			newOut := &accountOutput{
				rawOutput: *out,
				AccountID: cp.AccountID,
				Address:   cp.Address,
				keyIndex:  cp.KeyIndex,
			}
			result = append(result, newOut)
		}
	}

	return result
}

// upsertConfirmedAccountOutputs records the account data for confirmed utxos.
// If the account utxo already exists (because it's from a local tx), the
// block confirmation data will in the row will be updated.
func upsertConfirmedAccountOutputs(outs []*accountOutput, batch db.Batch) error {
	var u *account.UTXO

	for _, out := range outs {
		u = &account.UTXO{
			OutputID:            out.OutputID,
			SourceID:            out.sourceID,
			AssetID:             *out.AssetId,
			Amount:              out.Amount,
			SourcePos:           out.sourcePos,
			ControlProgram:      out.ControlProgram,
			ControlProgramIndex: out.keyIndex,
			AccountID:           out.AccountID,
			Address:             out.Address,
			ValidHeight:         out.ValidHeight,
		}

		data, err := json.Marshal(u)
		if err != nil {
			return errors.Wrap(err, "failed marshal accountutxo")
		}

		if segwit.IsP2WScript(out.ControlProgram) {
			// standard UTXOs
			batch.Set(account.StandardUTXOKey(out.OutputID), data)
		} else {
			// contract UTXOs
			batch.Set(account.ContractUTXOKey(out.OutputID), data)
		}

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
			if bytes := w.DB.Get(account.CPKey(hash)); bytes != nil {
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
	formatKey := w.DB.Get(calcTxIndexKey(txID))
	if formatKey == nil {
		return nil, fmt.Errorf("No transaction(tx_id=%s) ", txID)
	}

	annotatedTx := &query.AnnotatedTx{}
	txInfo := w.DB.Get(calcAnnotatedKey(string(formatKey)))
	if err := json.Unmarshal(txInfo, annotatedTx); err != nil {
		return nil, err
	}

	return annotatedTx, nil
}

// GetTransactionsByTxID get account txs by account tx ID
func (w *Wallet) GetTransactionsByTxID(txID string) ([]*query.AnnotatedTx, error) {
	var annotatedTxs []*query.AnnotatedTx
	formatKey := ""

	if txID != "" {
		rawFormatKey := w.DB.Get(calcTxIndexKey(txID))
		if rawFormatKey == nil {
			return nil, fmt.Errorf("No transaction(txid=%s) ", txID)
		}
		formatKey = string(rawFormatKey)
	}

	txIter := w.DB.IteratorPrefix(calcAnnotatedKey(formatKey))
	defer txIter.Release()
	for txIter.Next() {
		annotatedTx := &query.AnnotatedTx{}
		if err := json.Unmarshal(txIter.Value(), annotatedTx); err != nil {
			return nil, err
		}
		annotatedTxs = append([]*query.AnnotatedTx{annotatedTx}, annotatedTxs...)
	}

	return annotatedTxs, nil
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

// GetTransactionsByAccountID get account txs by account ID
func (w *Wallet) GetTransactionsByAccountID(accountID string) ([]*query.AnnotatedTx, error) {
	var annotatedTxs []*query.AnnotatedTx

	txIter := w.DB.IteratorPrefix([]byte(TxPrefix))
	defer txIter.Release()
	for txIter.Next() {
		annotatedTx := &query.AnnotatedTx{}
		if err := json.Unmarshal(txIter.Value(), &annotatedTx); err != nil {
			return nil, err
		}

		if findTransactionsByAccount(annotatedTx, accountID) {
			annotatedTxs = append(annotatedTxs, annotatedTx)
		}
	}

	return annotatedTxs, nil
}

// GetAccountUTXOs return all account unspent outputs
func (w *Wallet) GetAccountUTXOs(id string) []account.UTXO {
	var accountUTXOs []account.UTXO

	accountUTXOIter := w.DB.IteratorPrefix([]byte(account.UTXOPreFix + id))
	defer accountUTXOIter.Release()
	for accountUTXOIter.Next() {
		accountUTXO := account.UTXO{}
		if err := json.Unmarshal(accountUTXOIter.Value(), &accountUTXO); err != nil {
			hashKey := accountUTXOIter.Key()[len(account.UTXOPreFix):]
			log.WithField("UTXO hash", string(hashKey)).Warn("get account UTXO")
		} else {
			accountUTXOs = append(accountUTXOs, accountUTXO)
		}
	}

	return accountUTXOs
}

// GetAccountBalances return all account balances
func (w *Wallet) GetAccountBalances(id string) []AccountBalance {
	return w.indexBalances(w.GetAccountUTXOs(""))
}

// AccountBalance account balance
type AccountBalance struct {
	AccountID  string `json:"account_id"`
	Alias      string `json:"account_alias"`
	AssetAlias string `json:"asset_alias"`
	AssetID    string `json:"asset_id"`
	Amount     uint64 `json:"amount"`
}

func (w *Wallet) indexBalances(accountUTXOs []account.UTXO) []AccountBalance {
	accBalance := make(map[string]map[string]uint64)
	balances := make([]AccountBalance, 0)
	tmpBalance := AccountBalance{}

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
			assetAlias := w.AssetReg.GetAliasByID(assetID)
			tmpBalance.Alias = alias
			tmpBalance.AccountID = id
			tmpBalance.AssetID = assetID
			tmpBalance.AssetAlias = assetAlias
			tmpBalance.Amount = accBalance[id][assetID]
			balances = append(balances, tmpBalance)
		}
	}

	return balances
}
