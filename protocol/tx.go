package protocol

import (
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/validation"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// ValidateTx validates the given transaction. A cache holds
// per-transaction validation results and is consulted before
// performing full validation.
func (c *Chain) ValidateTx(tx *legacy.Tx) error {
	newTx := tx.Tx
	if ok := c.txPool.HaveTransaction(&newTx.ID); ok {
		return c.txPool.GetErrCache(&newTx.ID)
	}

	oldBlock, err := c.GetBlockByHash(c.state.hash)
	if err != nil {
		return err
	}
	block := legacy.MapBlock(oldBlock)
	fee, err := validation.ValidateTx(newTx, block)

	if err != nil {
		c.txPool.AddErrCache(&newTx.ID, err)
		return err
	}

	c.txPool.AddTransaction(tx, block.BlockHeader.Height, fee)
	return errors.Sub(ErrBadTx, err)
}
