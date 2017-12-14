package state

import (
	"errors"

	"github.com/bytom/blockchain/txdb/storage"
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

func (view *UtxoViewpoint) HasUtxo(hash *bc.Hash) bool {
	_, ok := view.Entries[*hash]
	return ok
}

func (view *UtxoViewpoint) ApplyTransaction(block *bc.Block, tx *bc.Tx) error {
	for _, prevout := range tx.SpentOutputIDs {
		entry, ok := view.Entries[prevout]
		if !ok {
			return errors.New("fail to find utxo entry")
		}
		if entry.Spend {
			return errors.New("utxo has been spend")
		}
		entry.SpendOutput()
	}

	for _, id := range tx.TxHeader.ResultIds {
		e := tx.Entries[*id]
		if _, ok := e.(*bc.Output); !ok {
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

func (view *UtxoViewpoint) DetachTransaction(tx *bc.Tx) error {
	for _, prevout := range tx.SpentOutputIDs {
		entry, ok := view.Entries[prevout]
		if !ok || !entry.Spend {
			return errors.New("try to revert a unspend utxo")
		}

		if !ok {
			view.Entries[prevout] = storage.NewUtxoEntry(false, 0, false)
			continue
		}

		entry.UnspendOutput()
	}

	for _, id := range tx.TxHeader.ResultIds {
		e := tx.Entries[*id]
		if _, ok := e.(*bc.Output); !ok {
			continue
		}

		view.Entries[*id] = storage.NewUtxoEntry(false, 0, true)
	}
	return nil
}

func (view *UtxoViewpoint) DetachBlock(block *bc.Block) error {
	for _, tx := range block.Transactions {
		if err := view.DetachTransaction(tx); err != nil {
			return err
		}
	}
	return nil
}
