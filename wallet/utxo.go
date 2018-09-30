package wallet

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/consensus"
	"github.com/bytom/consensus/segwit"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// GetAccountUtxos return all account unspent outputs
func (w *Wallet) GetAccountUtxos(accountID string, id string, unconfirmed, isSmartContract bool) []*account.UTXO {
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
			log.WithField("err", err).Warn("GetAccountUtxos fail on unmarshal utxo")
			continue
		}

		if accountID == accountUtxo.AccountID || accountID == "" {
			accountUtxos = append(accountUtxos, accountUtxo)
		}
	}
	return accountUtxos
}

func (w *Wallet) attachUtxos(batch db.Batch, b *types.Block, txStatus *bc.TransactionStatus) {
	for txIndex, tx := range b.Transactions {
		statusFail, err := txStatus.GetStatus(txIndex)
		if err != nil {
			log.WithField("err", err).Error("attachUtxos fail on get tx status")
			continue
		}

		//hand update the transaction input utxos
		inputUtxos := txInToUtxos(tx, statusFail)
		for _, inputUtxo := range inputUtxos {
			if segwit.IsP2WScript(inputUtxo.ControlProgram) {
				batch.Delete(account.StandardUTXOKey(inputUtxo.OutputID))
			} else {
				batch.Delete(account.ContractUTXOKey(inputUtxo.OutputID))
			}
		}

		//hand update the transaction output utxos
		validHeight := uint64(0)
		if txIndex == 0 {
			validHeight = b.Height + consensus.CoinbasePendingBlockNumber
		}
		outputUtxos := txOutToUtxos(tx, statusFail, validHeight)
		utxos := w.filterAccountUtxo(outputUtxos)
		if err := batchSaveUtxos(utxos, batch); err != nil {
			log.WithField("err", err).Error("attachUtxos fail on batchSaveUtxos")
		}
	}
}

func (w *Wallet) detachUtxos(batch db.Batch, b *types.Block, txStatus *bc.TransactionStatus) {
	for txIndex := len(b.Transactions) - 1; txIndex >= 0; txIndex-- {
		tx := b.Transactions[txIndex]
		for j := range tx.Outputs {
			resOut, err := tx.Output(*tx.ResultIds[j])
			if err != nil {
				continue
			}

			if segwit.IsP2WScript(resOut.ControlProgram.Code) {
				batch.Delete(account.StandardUTXOKey(*tx.ResultIds[j]))
			} else {
				batch.Delete(account.ContractUTXOKey(*tx.ResultIds[j]))
			}
		}

		statusFail, err := txStatus.GetStatus(txIndex)
		if err != nil {
			log.WithField("err", err).Error("detachUtxos fail on get tx status")
			continue
		}

		inputUtxos := txInToUtxos(tx, statusFail)
		utxos := w.filterAccountUtxo(inputUtxos)
		if err := batchSaveUtxos(utxos, batch); err != nil {
			log.WithField("err", err).Error("detachUtxos fail on batchSaveUtxos")
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
			for _, utxo := range outsByScript[s] {
				result = append(result, utxo)
			}
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
			log.WithField("err", err).Error("filterAccountUtxo fail on unmarshal control program")
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

func batchSaveUtxos(utxos []*account.UTXO, batch db.Batch) error {
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

func txInToUtxos(tx *types.Tx, statusFail bool) []*account.UTXO {
	utxos := []*account.UTXO{}
	for _, inpID := range tx.Tx.InputIDs {
		sp, err := tx.Spend(inpID)
		if err != nil {
			continue
		}

		resOut, err := tx.Output(*sp.SpentOutputId)
		if err != nil {
			log.WithField("err", err).Error("txInToUtxos fail on get resOut")
			continue
		}

		if statusFail && *resOut.Source.Value.AssetId != *consensus.BTMAssetID {
			continue
		}

		utxos = append(utxos, &account.UTXO{
			OutputID:       *sp.SpentOutputId,
			AssetID:        *resOut.Source.Value.AssetId,
			Amount:         resOut.Source.Value.Amount,
			ControlProgram: resOut.ControlProgram.Code,
			SourceID:       *resOut.Source.Ref,
			SourcePos:      resOut.Source.Position,
		})
	}
	return utxos
}

func txOutToUtxos(tx *types.Tx, statusFail bool, vaildHeight uint64) []*account.UTXO {
	utxos := []*account.UTXO{}
	for i, out := range tx.Outputs {
		bcOut, err := tx.Output(*tx.ResultIds[i])
		if err != nil {
			continue
		}

		if statusFail && *out.AssetAmount.AssetId != *consensus.BTMAssetID {
			continue
		}

		utxos = append(utxos, &account.UTXO{
			OutputID:       *tx.OutputID(i),
			AssetID:        *out.AssetAmount.AssetId,
			Amount:         out.Amount,
			ControlProgram: out.ControlProgram,
			SourceID:       *bcOut.Source.Ref,
			SourcePos:      bcOut.Source.Position,
			ValidHeight:    vaildHeight,
		})
	}
	return utxos
}
