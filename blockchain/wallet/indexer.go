package wallet

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/account"
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
	keyIndex  uint64
	change    bool
}

const (
	//TxPreFix is wallet database transactions prefix
	TxPreFix = "TXS:"
)

func calcAnnotatedKey(blockHeight uint64, position uint32) []byte {
	return []byte(fmt.Sprintf("%s%016x%08x", TxPreFix, blockHeight, position))
}

func calcDeletePreFix(blockHeight uint64) []byte {
	return []byte(fmt.Sprintf("%s%016x", TxPreFix, blockHeight))
}

//deleteTransaction delete transactions when orphan block rollback
func deleteTransactions(batch *db.Batch, height uint64, b *legacy.Block, w *Wallet) {
	txIter := w.DB.IteratorPrefix(calcDeletePreFix(height))
	defer txIter.Release()

	for txIter.Next() {
		(*batch).Delete(txIter.Key())
	}
}

//ReverseAccountUTXOs process the invalid blocks when orphan block rollback
func reverseAccountUTXOs(batch *db.Batch, b *legacy.Block, w *Wallet) {
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
	if err = upsertConfirmedAccountOutputs(accOuts, b, batch, w); err != nil {
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
			(*batch).Delete(account.UTXOKey(*resOutID))
		}
	}
}

//indexTransactions saves all annotated transactions to the database.
func indexTransactions(batch *db.Batch, b *legacy.Block, w *Wallet) error {
	annotatedTxs := filterAccountTxs(b, w)
	annotateTxsAsset(annotatedTxs, w.DB)
	annotateTxsAccount(annotatedTxs, w.DB)

	for pos, tx := range annotatedTxs {
		rawTx, err := json.MarshalIndent(tx, "", "    ")
		if err != nil {
			return errors.Wrap(err, "inserting annotated_txs to db")
		}

		(*batch).Set(calcAnnotatedKey(b.Height, uint32(pos)), rawTx)
	}

	return nil
}

//buildAccountUTXOs process valid blocks to build account unspent outputs db
func buildAccountUTXOs(batch *db.Batch, b *legacy.Block, w *Wallet) {
	var err error

	//handle spent UTXOs
	delOutputIDs := prevoutDBKeys(b.Transactions...)
	for _, delOutputID := range delOutputIDs {
		(*batch).Delete(account.UTXOKey(delOutputID))
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

	if err = upsertConfirmedAccountOutputs(accOuts, b, batch, w); err != nil {
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
func upsertConfirmedAccountOutputs(outs []*accountOutput, block *legacy.Block, batch *db.Batch, w *Wallet) error {
	var u *account.UTXO

	for _, out := range outs {
		u = &account.UTXO{OutputID: out.OutputID.Bytes(),
			AssetID:      out.AssetId.Bytes(),
			Amount:       out.Amount,
			AccountID:    out.AccountID,
			ProgramIndex: out.keyIndex,
			Program:      out.ControlProgram,
			SourceID:     out.sourceID.Bytes(),
			SourcePos:    out.sourcePos,
			RefData:      out.refData.Bytes(),
			Change:       out.change}

		rawUTXO, err := json.Marshal(u)
		if err != nil {
			return errors.Wrap(err, "failed marshal accountutxo")
		}

		(*batch).Set(account.UTXOKey(out.OutputID), rawUTXO)
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
