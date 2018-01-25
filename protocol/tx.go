package protocol

import (
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/validation"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")
var ErrTimeLimit = errors.New("invalid transaction maxtime")

// ValidateTx validates the given transaction. A cache holds
// per-transaction validation results and is consulted before
// performing full validation.
func (c *Chain) ValidateTx(tx *legacy.Tx) error {
	newTx := tx.Tx
	if ok := c.txPool.HaveTransaction(&newTx.ID); ok {
		return c.txPool.GetErrCache(&newTx.ID)
	}

	oldBlock := c.BestBlock()
	if tx.MaxTime > oldBlock.Timestamp {
		c.txPool.AddErrCache(&newTx.ID, ErrTimeLimit)
		return ErrTimeLimit
	}

	// validate the BVM contract
	gasOnlyTx := false
	block := legacy.MapBlock(oldBlock)
	fee, gasVaild, err := validation.ValidateTx(newTx, block)
	if err != nil {
		if !gasVaild {
			c.txPool.AddErrCache(&newTx.ID, err)
			return err
		}
		gasOnlyTx = true
	}

	// validate the UTXO
	view := c.txPool.GetTransactionUTXO(tx.Tx)
	if err := c.GetTransactionsUtxo(view, []*bc.Tx{newTx}); err != nil {
		c.txPool.AddErrCache(&newTx.ID, err)
		return err
	}

	if err := view.ApplyTransaction(block, newTx, gasOnlyTx); err != nil {
		c.txPool.AddErrCache(&newTx.ID, err)
		return err
	}

	c.txPool.AddTransaction(tx, view, block.BlockHeader.Height, fee)
	return nil
}
