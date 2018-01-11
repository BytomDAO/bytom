package wallet

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

type rawOutput struct {
	OutputID bc.Hash
	bc.AssetAmount
	ControlProgram []byte
	txHash         bc.Hash
	outputIndex    uint32
	sourceID       bc.Hash
	sourcePos      uint64
	refData        bc.Hash
}

type accountOutput struct {
	rawOutput
	AccountID string
	Address   string
	keyIndex  uint64
	change    bool
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

//deleteTransaction delete transactions when orphan block rollback
func (w *Wallet) deleteTransactions(batch db.Batch, height uint64) {
	tmpTx := query.AnnotatedTx{}

	txIter := w.DB.IteratorPrefix(calcDeleteKey(height))
	defer txIter.Release()

	for txIter.Next() {
		if err := json.Unmarshal(txIter.Value(), &tmpTx); err == nil {
			//delete index
			batch.Delete(calcTxIndexKey(tmpTx.ID.String()))
		}

		batch.Delete(txIter.Key())
	}
}

//ReverseAccountUTXOs process the invalid blocks when orphan block rollback
func (w *Wallet) reverseAccountUTXOs(batch db.Batch, b *legacy.Block) {
	var err error

	//unknow how many spent and retire outputs
	reverseOuts := make([]*rawOutput, 0)

	//handle spent UTXOs
	for _, tx := range b.Transactions {
		for _, inpID := range tx.Tx.InputIDs {
			//spend and retire
			sp, err := tx.Spend(inpID)
			if err != nil {
				continue
			}

			resOut, ok := tx.Entries[*sp.SpentOutputId].(*bc.Output)
			if !ok {
				continue
			}

			out := &rawOutput{
				OutputID:       *sp.SpentOutputId,
				AssetAmount:    *resOut.Source.Value,
				ControlProgram: resOut.ControlProgram.Code,
				txHash:         tx.ID,
				sourceID:       *resOut.Source.Ref,
				sourcePos:      resOut.Source.Position,
				refData:        *resOut.Data,
			}
			reverseOuts = append(reverseOuts, out)
		}
	}

	accOuts := loadAccountInfo(reverseOuts, w)
	if err = upsertConfirmedAccountOutputs(accOuts, batch); err != nil {
		log.WithField("err", err).Error("reversing account spent and retire outputs")
		return
	}

	//handle new UTXOs
	for _, tx := range b.Transactions {
		for j := range tx.Outputs {
			resOutID := tx.ResultIds[j]
			if _, ok := tx.Entries[*resOutID].(*bc.Output); !ok {
				//retirement
				continue
			}
			//delete new UTXOs
			batch.Delete(account.UTXOKey(*resOutID))
		}
	}
}

//save external assets definition
func saveExternalAssetDefinition(b *legacy.Block, walletDB db.DB) {
	storeBatch := walletDB.NewBatch()
	defer storeBatch.Write()

	for _, tx := range b.Transactions {
		for _, orig := range tx.Inputs {
			if ii, ok := orig.TypedInput.(*legacy.IssuanceInput); ok {
				if isValidJSON(ii.AssetDefinition) {
					assetID := ii.AssetID()
					storeBatch.Set(asset.CalcExtAssetKey(&assetID), ii.AssetDefinition)
				}
			}
		}
	}
}

//indexTransactions saves all annotated transactions to the database.
func (w *Wallet) indexTransactions(batch db.Batch, b *legacy.Block) error {
	annotatedTxs := filterAccountTxs(b, w)
	saveExternalAssetDefinition(b, w.DB)
	annotateTxsAsset(annotatedTxs, w.DB)
	annotateTxsAccount(annotatedTxs, w.DB)

	for _, tx := range annotatedTxs {
		rawTx, err := json.Marshal(tx)
		if err != nil {
			return errors.Wrap(err, "inserting annotated_txs to db")
		}

		batch.Set(calcAnnotatedKey(formatKey(b.Height, uint32(tx.Position))), rawTx)
		batch.Set(calcTxIndexKey(tx.ID.String()), []byte(formatKey(b.Height, uint32(tx.Position))))
	}
	return nil
}

//buildAccountUTXOs process valid blocks to build account unspent outputs db
func (w *Wallet) buildAccountUTXOs(batch db.Batch, b *legacy.Block) {
	var err error

	//handle spent UTXOs
	delOutputIDs := prevoutDBKeys(b.Transactions...)
	for _, delOutputID := range delOutputIDs {
		batch.Delete(account.UTXOKey(delOutputID))
	}

	//handle new UTXOs
	outs := make([]*rawOutput, 0, len(b.Transactions))
	for _, tx := range b.Transactions {
		for j, out := range tx.Outputs {
			resOutID := tx.ResultIds[j]
			resOut, ok := tx.Entries[*resOutID].(*bc.Output)
			if !ok {
				continue
			}
			out := &rawOutput{
				OutputID:       *tx.OutputID(j),
				AssetAmount:    out.AssetAmount,
				ControlProgram: out.ControlProgram,
				txHash:         tx.ID,
				outputIndex:    uint32(j),
				sourceID:       *resOut.Source.Ref,
				sourcePos:      resOut.Source.Position,
				refData:        *resOut.Data,
			}
			outs = append(outs, out)
		}
	}
	accOuts := loadAccountInfo(outs, w)

	if err = upsertConfirmedAccountOutputs(accOuts, batch); err != nil {
		log.WithField("err", err).Error("building new account outputs")
		return
	}
}

func prevoutDBKeys(txs ...*legacy.Tx) (outputIDs []bc.Hash) {
	for _, tx := range txs {
		for _, inpID := range tx.Tx.InputIDs {
			if sp, err := tx.Spend(inpID); err == nil {
				outputIDs = append(outputIDs, *sp.SpentOutputId)
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
				change:    cp.Change,
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
			RefDataHash:         out.refData,
			ControlProgramIndex: out.keyIndex,
			AccountID:           out.AccountID,
			Address:             out.Address,
		}

		data, err := json.Marshal(u)
		if err != nil {
			return errors.Wrap(err, "failed marshal accountutxo")
		}
		batch.Set(account.UTXOKey(out.OutputID), data)
	}
	return nil
}

// filt related and build the fully annotated transactions.
func filterAccountTxs(b *legacy.Block, w *Wallet) []*query.AnnotatedTx {
	annotatedTxs := make([]*query.AnnotatedTx, 0, len(b.Transactions))
	for pos, tx := range b.Transactions {
		local := false
		for _, v := range tx.Outputs {
			var hash [32]byte

			sha3pool.Sum256(hash[:], v.ControlProgram)
			if bytes := w.DB.Get(account.CPKey(hash)); bytes != nil {
				annotatedTxs = append(annotatedTxs, buildAnnotatedTransaction(tx, b, uint32(pos)))
				local = true
				break
			}
		}

		if local == true {
			continue
		}

		for _, v := range tx.Inputs {
			outid, err := v.SpentOutputID()
			if err != nil {
				continue
			}
			if bytes := w.DB.Get(account.UTXOKey(outid)); bytes != nil {
				annotatedTxs = append(annotatedTxs, buildAnnotatedTransaction(tx, b, uint32(pos)))
				break
			}
		}
	}

	return annotatedTxs
}

//GetTransactionsByTxID get account txs by account tx ID
func (w *Wallet) GetTransactionsByTxID(txID string) ([]query.AnnotatedTx, error) {
	annotatedTx := query.AnnotatedTx{}
	annotatedTxs := make([]query.AnnotatedTx, 0)
	formatKey := ""

	if txID != "" {
		rawFormatKey := w.DB.Get(calcTxIndexKey(txID))
		if rawFormatKey == nil {
			return nil, fmt.Errorf("No transaction(txid=%s)", txID)
		}
		formatKey = string(rawFormatKey)
	}

	txIter := w.DB.IteratorPrefix([]byte(TxPrefix + formatKey))
	defer txIter.Release()
	for txIter.Next() {
		if err := json.Unmarshal(txIter.Value(), &annotatedTx); err != nil {
			return nil, err
		}
		annotatedTxs = append(annotatedTxs, annotatedTx)
	}

	return annotatedTxs, nil
}

func findTransactionsByAccount(annotatedTx query.AnnotatedTx, accountID string) bool {
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

//GetTransactionsByAccountID get account txs by account ID
func (w *Wallet) GetTransactionsByAccountID(accountID string) ([]query.AnnotatedTx, error) {
	annotatedTxs := make([]query.AnnotatedTx, 0)

	txIter := w.DB.IteratorPrefix([]byte(TxPrefix))
	defer txIter.Release()
	for txIter.Next() {
		annotatedTx := query.AnnotatedTx{}
		if err := json.Unmarshal(txIter.Value(), &annotatedTx); err != nil {
			return nil, err
		}

		if findTransactionsByAccount(annotatedTx, accountID) {
			annotatedTxs = append(annotatedTxs, annotatedTx)
		}
	}

	return annotatedTxs, nil
}

//GetAccountUTXOs return all account unspent outputs
func (w *Wallet) GetAccountUTXOs(id string) ([]account.UTXO, error) {
	accountUTXO := account.UTXO{}
	accountUTXOs := make([]account.UTXO, 0)

	accountUTXOIter := w.DB.IteratorPrefix([]byte(account.UTXOPreFix + id))
	defer accountUTXOIter.Release()
	for accountUTXOIter.Next() {
		if err := json.Unmarshal(accountUTXOIter.Value(), &accountUTXO); err != nil {
			hashKey := accountUTXOIter.Key()[len(account.UTXOPreFix):]
			log.WithField("UTXO hash", string(hashKey)).Warn("get account UTXO")
			continue
		}

		accountUTXOs = append(accountUTXOs, accountUTXO)
	}

	return accountUTXOs, nil
}
