package wallet

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/account"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/consensus/segwit"
	"github.com/bytom/bytom/crypto/sha3pool"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

// GetAccountUtxos return all account unspent outputs
func (w *Wallet) GetAccountUtxos(accountID string, id string, unconfirmed, isSmartContract, vote bool) []*account.UTXO {
	prefix := account.UTXOPreFix
	if isSmartContract {
		prefix = account.SUTXOPrefix
	}

	accountUtxos := []*account.UTXO{}
	if unconfirmed {
		accountUtxos = w.AccountMgr.ListUnconfirmedUtxo(accountID, isSmartContract)
	}

	accountUtxoIter := w.DB.IteratorPrefix([]byte(prefix + id))
	defer accountUtxoIter.Release()

	for accountUtxoIter.Next() {
		accountUtxo := &account.UTXO{}
		if err := json.Unmarshal(accountUtxoIter.Value(), accountUtxo); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warn("GetAccountUtxos fail on unmarshal utxo")
			continue
		}

		if vote && accountUtxo.Vote == nil {
			continue
		}

		if accountID == accountUtxo.AccountID || accountID == "" {
			accountUtxos = append(accountUtxos, accountUtxo)
		}
	}
	return accountUtxos
}

func (w *Wallet) attachUtxos(batch dbm.Batch, b *types.Block) {
	for _, tx := range b.Transactions {
		// hand update the transaction input utxos
		inputUtxos := txInToUtxos(tx)
		for _, inputUtxo := range inputUtxos {
			if segwit.IsP2WScript(inputUtxo.ControlProgram) {
				batch.Delete(account.StandardUTXOKey(inputUtxo.OutputID))
			} else {
				batch.Delete(account.ContractUTXOKey(inputUtxo.OutputID))
			}
		}

		// hand update the transaction output utxos
		outputUtxos := txOutToUtxos(tx, b.Height)
		utxos := w.filterAccountUtxo(outputUtxos)
		if err := batchSaveUtxos(utxos, batch); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("attachUtxos fail on batchSaveUtxos")
		}
	}
}

func (w *Wallet) detachUtxos(batch dbm.Batch, b *types.Block) {
	for txIndex := len(b.Transactions) - 1; txIndex >= 0; txIndex-- {
		tx := b.Transactions[txIndex]
		for j := range tx.Outputs {
			resOut, err := tx.OriginalOutput(*tx.ResultIds[j])
			if err != nil {
				continue
			}

			if segwit.IsP2WScript(resOut.ControlProgram.Code) {
				batch.Delete(account.StandardUTXOKey(*tx.ResultIds[j]))
			} else {
				batch.Delete(account.ContractUTXOKey(*tx.ResultIds[j]))
			}
		}

		inputUtxos := txInToUtxos(tx)
		utxos := w.filterAccountUtxo(inputUtxos)
		if err := batchSaveUtxos(utxos, batch); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("detachUtxos fail on batchSaveUtxos")
			return
		}
	}
}

func (w *Wallet) filterAccountUtxo(utxos []*account.UTXO) []*account.UTXO {
	outsByScript := make(map[string][]*account.UTXO, len(utxos))
	for _, utxo := range utxos {
		scriptStr := string(utxo.ControlProgram)
		outsByScript[scriptStr] = append(outsByScript[scriptStr], utxo)
	}

	result := make([]*account.UTXO, 0, len(utxos))
	for s := range outsByScript {
		if !segwit.IsP2WScript([]byte(s)) {
			continue
		}

		var hash [32]byte
		sha3pool.Sum256(hash[:], []byte(s))
		data := w.DB.Get(account.ContractKey(hash))
		if data == nil {
			continue
		}

		cp := &account.CtrlProgram{}
		if err := json.Unmarshal(data, cp); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("filterAccountUtxo fail on unmarshal control program")
			continue
		}

		for _, utxo := range outsByScript[s] {
			utxo.AccountID = cp.AccountID
			utxo.Address = cp.Address
			utxo.ControlProgramIndex = cp.KeyIndex
			utxo.Change = cp.Change
			result = append(result, utxo)
		}
	}
	return result
}

func batchSaveUtxos(utxos []*account.UTXO, batch dbm.Batch) error {
	for _, utxo := range utxos {
		data, err := json.Marshal(utxo)
		if err != nil {
			return errors.Wrap(err, "failed marshal accountutxo")
		}

		if segwit.IsP2WScript(utxo.ControlProgram) {
			batch.Set(account.StandardUTXOKey(utxo.OutputID), data)
		} else {
			batch.Set(account.ContractUTXOKey(utxo.OutputID), data)
		}
	}
	return nil
}

func txInToUtxos(tx *types.Tx) []*account.UTXO {
	utxos := []*account.UTXO{}
	for _, inpID := range tx.Tx.InputIDs {
		var utxo *account.UTXO
		e, ok := tx.Entries[inpID]
		if !ok {
			continue
		}

		switch inp := e.(type) {
		case *bc.Spend:
			resOut, err := tx.OriginalOutput(*inp.SpentOutputId)
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("txInToUtxos fail on get resOut")
				continue
			}

			utxo = &account.UTXO{
				OutputID:       *inp.SpentOutputId,
				AssetID:        *resOut.Source.Value.AssetId,
				Amount:         resOut.Source.Value.Amount,
				ControlProgram: resOut.ControlProgram.Code,
				SourceID:       *resOut.Source.Ref,
				SourcePos:      resOut.Source.Position,
			}
		case *bc.VetoInput:
			resOut, err := tx.VoteOutput(*inp.SpentOutputId)
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("txInToUtxos fail on get resOut for vetoInput")
				continue
			}
			if *resOut.Source.Value.AssetId != *consensus.BTMAssetID {
				continue
			}
			utxo = &account.UTXO{
				OutputID:       *inp.SpentOutputId,
				AssetID:        *resOut.Source.Value.AssetId,
				Amount:         resOut.Source.Value.Amount,
				ControlProgram: resOut.ControlProgram.Code,
				SourceID:       *resOut.Source.Ref,
				SourcePos:      resOut.Source.Position,
				Vote:           resOut.Vote,
			}
		default:
			continue
		}
		utxos = append(utxos, utxo)
	}
	return utxos
}

func txOutToUtxos(tx *types.Tx, blockHeight uint64) []*account.UTXO {
	utxos := []*account.UTXO{}
	for i, out := range tx.Outputs {
		validHeight := uint64(0)
		entryOutput, ok := tx.Entries[*tx.ResultIds[i]]
		if !ok {
			log.WithFields(log.Fields{"module": logModule}).Error("txOutToUtxos fail on get entryOutput")
			continue
		}

		var utxo *account.UTXO
		switch bcOut := entryOutput.(type) {
		case *bc.OriginalOutput:
			if out.AssetAmount.Amount == uint64(0) {
				continue
			}

			if tx.Inputs[0].InputType() == types.CoinbaseInputType {
				validHeight = blockHeight + consensus.CoinbasePendingBlockNumber
			}

			utxo = &account.UTXO{
				OutputID:       *tx.OutputID(i),
				AssetID:        *out.AssetAmount.AssetId,
				Amount:         out.AssetAmount.Amount,
				ControlProgram: out.ControlProgram,
				SourceID:       *bcOut.Source.Ref,
				SourcePos:      bcOut.Source.Position,
				ValidHeight:    validHeight,
			}

		case *bc.VoteOutput:
			voteValidHeight := blockHeight + consensus.VotePendingBlockNums(blockHeight)
			if validHeight < voteValidHeight {
				validHeight = voteValidHeight
			}

			utxo = &account.UTXO{
				OutputID:       *tx.OutputID(i),
				AssetID:        *out.AssetAmount.AssetId,
				Amount:         out.AssetAmount.Amount,
				ControlProgram: out.ControlProgram,
				SourceID:       *bcOut.Source.Ref,
				SourcePos:      bcOut.Source.Position,
				ValidHeight:    validHeight,
				Vote:           bcOut.Vote,
			}

		default:
			log.WithFields(log.Fields{"module": logModule}).Warn("txOutToUtxos fail on get bcOut")
			continue
		}
		utxos = append(utxos, utxo)
	}
	return utxos
}
