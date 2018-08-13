package state

import (
	"errors"

	"github.com/bytom/consensus"
	"github.com/bytom/database/storage"
	"github.com/bytom/protocol/bc"
)

// UtxoViewpoint represents a view into the set of unspent transaction outputs
type UtxoViewpoint struct {
	Entries map[bc.Hash]*storage.UtxoEntry
}

// NewUtxoViewpoint returns a new empty unspent transaction output view.
func NewUtxoViewpoint() *UtxoViewpoint {
	return &UtxoViewpoint{
		Entries: make(map[bc.Hash]*storage.UtxoEntry),
	}
}

func (view *UtxoViewpoint) ApplyTransaction(block *bc.Block, tx *bc.Tx, statusFail bool) error {
	for _, prevout := range tx.SpentOutputIDs {
		spentOutput, err := tx.Output(prevout)
		if err != nil {
			return err
		}
		if statusFail && *spentOutput.Source.Value.AssetId != *consensus.BTMAssetID {
			continue
		}

		entry, ok := view.Entries[prevout]
		if !ok {
			return errors.New("fail to find utxo entry")
		}
		if entry.Spent {
			return errors.New("utxo has been spent")
		}
		if entry.IsCoinBase && entry.BlockHeight+consensus.CoinbasePendingBlockNumber > block.Height {
			return errors.New("coinbase utxo is not ready for use")
		}
		entry.SpendOutput()
	}

	for _, id := range tx.TxHeader.ResultIds {
		output, err := tx.Output(*id)
		if err != nil {
			// error due to it's a retirement, utxo doesn't care this output type so skip it
			continue
		}
		if statusFail && *output.Source.Value.AssetId != *consensus.BTMAssetID {
			continue
		}

		isCoinbase := false
		if block != nil && len(block.Transactions) > 0 && block.Transactions[0].ID == tx.ID {
			isCoinbase = true
		}
		view.Entries[*id] = storage.NewUtxoEntry(isCoinbase, block.Height, false)
	}
	return nil
}

func (view *UtxoViewpoint) ApplyBlock(block *bc.Block, txStatus *bc.TransactionStatus) error {
	for i, tx := range block.Transactions {
		statusFail, err := txStatus.GetStatus(i)
		if err != nil {
			return err
		}
		if err := view.ApplyTransaction(block, tx, statusFail); err != nil {
			return err
		}
	}
	return nil
}

func (view *UtxoViewpoint) CanSpend(hash *bc.Hash) bool {
	entry := view.Entries[*hash]
	return entry != nil && !entry.Spent
}

func (view *UtxoViewpoint) DetachTransaction(tx *bc.Tx, statusFail bool) error {
	for _, prevout := range tx.SpentOutputIDs {
		spentOutput, err := tx.Output(prevout)
		if err != nil {
			return err
		}
		if statusFail && *spentOutput.Source.Value.AssetId != *consensus.BTMAssetID {
			continue
		}

		entry, ok := view.Entries[prevout]
		if ok && !entry.Spent {
			return errors.New("try to revert an unspent utxo")
		}
		if !ok {
			view.Entries[prevout] = storage.NewUtxoEntry(false, 0, false)
			continue
		}
		entry.UnspendOutput()
	}

	for _, id := range tx.TxHeader.ResultIds {
		output, err := tx.Output(*id)
		if err != nil {
			// error due to it's a retirement, utxo doesn't care this output type so skip it
			continue
		}
		if statusFail && *output.Source.Value.AssetId != *consensus.BTMAssetID {
			continue
		}

		view.Entries[*id] = storage.NewUtxoEntry(false, 0, true)
	}
	return nil
}

func (view *UtxoViewpoint) DetachBlock(block *bc.Block, txStatus *bc.TransactionStatus) error {
	for i, tx := range block.Transactions {
		statusFail, err := txStatus.GetStatus(i)
		if err != nil {
			return err
		}
		if err := view.DetachTransaction(tx, statusFail); err != nil {
			return err
		}
	}
	return nil
}

func (view *UtxoViewpoint) HasUtxo(hash *bc.Hash) bool {
	_, ok := view.Entries[*hash]
	return ok
}
