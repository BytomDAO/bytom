package state

import (
	"errors"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/protocol/bc"
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

func (view *UtxoViewpoint) ApplyTransaction(block *bc.Block, tx *bc.Tx) error {
	for _, prevout := range tx.SpentOutputIDs {
		_, err := tx.OriginalOutput(prevout)
		if err != nil {
			return err
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
		_, err := tx.OriginalOutput(*id)
		if err != nil {
			// error due to it's a retirement, utxo doesn't care this output type so skip it
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

func (view *UtxoViewpoint) ApplyBlock(block *bc.Block) error {
	for _, tx := range block.Transactions {
		if err := view.ApplyTransaction(block, tx); err != nil {
			return err
		}
	}
	return nil
}

func (view *UtxoViewpoint) CanSpend(hash *bc.Hash) bool {
	entry := view.Entries[*hash]
	return entry != nil && !entry.Spent
}

func (view *UtxoViewpoint) DetachTransaction(tx *bc.Tx) error {
	for _, prevout := range tx.SpentOutputIDs {
		_, err := tx.OriginalOutput(prevout)
		if err != nil {
			return err
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
		_, err := tx.OriginalOutput(*id)
		if err != nil {
			// error due to it's a retirement, utxo doesn't care this output type so skip it
			continue
		}

		view.Entries[*id] = storage.NewUtxoEntry(false, 0, true)
	}
	return nil
}

func (view *UtxoViewpoint) DetachBlock(block *bc.Block) error {
	for i := len(block.Transactions) - 1; i >= 0; i-- {
		if err := view.DetachTransaction(block.Transactions[i]); err != nil {
			return err
		}
	}
	return nil
}

func (view *UtxoViewpoint) HasUtxo(hash *bc.Hash) bool {
	_, ok := view.Entries[*hash]
	return ok
}
