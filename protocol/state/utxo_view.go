package state

import (
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/errors"
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
	if err := view.applySpendUtxo(block, tx); err != nil {
		return err
	}

	return view.applyOutputUtxo(block, tx)
}

func (view *UtxoViewpoint) applySpendUtxo(block *bc.Block, tx *bc.Tx) error {
	for _, prevout := range tx.SpentOutputIDs {
		entry, ok := view.Entries[prevout]
		if !ok {
			return errors.New("fail to find utxo entry")
		}
		if entry.Spent {
			return errors.New("utxo has been spent")
		}
		switch entry.Type {
		case storage.CoinbaseUTXOType:
			if entry.BlockHeight+consensus.CoinbasePendingBlockNumber > block.Height {
				return errors.New("coinbase utxo is not ready for use")
			}
		case storage.VoteUTXOType:
			if entry.BlockHeight + consensus.VotePendingBlockNums(block.Height) > block.Height {
				return errors.New("Coin is  within the voting lock time")
			}
		}

		entry.SpendOutput()
	}
	return nil
}

func (view *UtxoViewpoint) applyOutputUtxo(block *bc.Block, tx *bc.Tx) error {
	for _, id := range tx.TxHeader.ResultIds {
		entryOutput, ok := tx.Entries[*id]
		if !ok {
			// error due to it's a retirement, utxo doesn't care this output type so skip it
			continue
		}


		var utxoType uint32
		var amount uint64
		switch output := entryOutput.(type) {
		case *bc.OriginalOutput:
			amount = output.Source.Value.Amount
			utxoType = storage.NormalUTXOType
		case *bc.VoteOutput:
			amount = output.Source.Value.Amount
			utxoType = storage.VoteUTXOType
		default:
			// due to it's a retirement, utxo doesn't care this output type so skip it
			continue
		}

		if amount == 0 {
			continue
		}

		if block != nil && len(block.Transactions) > 0 && block.Transactions[0].ID == tx.ID {
			utxoType = storage.CoinbaseUTXOType
		}
		view.Entries[*id] = storage.NewUtxoEntry(utxoType, block.Height, false)
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
	if err := view.detachSpendUtxo(tx); err != nil {
		return err
	}

	return view.detachOutputUtxo(tx)
}

func (view *UtxoViewpoint) detachSpendUtxo(tx *bc.Tx) error {
	for _, prevout := range tx.SpentOutputIDs {
		entryOutput, ok := tx.Entries[prevout]
		if !ok {
			return errors.New("fail to find utxo entry")
		}

		var utxoType uint32
		switch entryOutput.(type) {
		case *bc.OriginalOutput:
			utxoType = storage.NormalUTXOType
		case *bc.VoteOutput:
			utxoType = storage.VoteUTXOType
		default:
			return errors.Wrapf(bc.ErrEntryType, "entry %x has unexpected type %T", prevout.Bytes(), entryOutput)
		}

		entry, ok := view.Entries[prevout]
		if ok && !entry.Spent {
			return errors.New("try to revert an unspent utxo")
		}
		if !ok {
			view.Entries[prevout] = storage.NewUtxoEntry(utxoType, 0, false)
			continue
		}
		entry.UnspendOutput()
	}
	return nil
}

func (view *UtxoViewpoint) detachOutputUtxo(tx *bc.Tx) error {
	for _, id := range tx.TxHeader.ResultIds {
		entryOutput, ok := tx.Entries[*id]
		if !ok {
			// error due to it's a retirement, utxo doesn't care this output type so skip it
			continue
		}

		var utxoType uint32
		var amount uint64
		switch output := entryOutput.(type) {
		case *bc.OriginalOutput:
			amount = output.Source.Value.Amount
			utxoType = storage.NormalUTXOType
		case *bc.VoteOutput:
			amount = output.Source.Value.Amount
			utxoType = storage.VoteUTXOType
		default:
			// due to it's a retirement, utxo doesn't care this output type so skip it
			continue
		}

		if amount == 0 {
			continue
		}

		view.Entries[*id] = storage.NewUtxoEntry(utxoType, 0, true)
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
