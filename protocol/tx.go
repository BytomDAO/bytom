package protocol

import (
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/validation"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// ValidateTx validates the given transaction. A cache holds
// per-transaction validation results and is consulted before
// performing full validation.
func (c *Chain) ValidateTx(tx *types.Tx) (bool, error) {
	newTx := tx.Tx
	bh := c.BestBlockHeader()
	block := types.MapBlock(&types.Block{BlockHeader: *bh})
	if ok := c.txPool.HaveTransaction(&newTx.ID); ok {
		return false, c.txPool.GetErrCache(&newTx.ID)
	}

	// validate the UTXO
	view := c.txPool.GetTransactionUTXO(tx.Tx)
	if err := c.GetTransactionsUtxo(view, []*bc.Tx{newTx}); err != nil {
		c.txPool.AddErrCache(&newTx.ID, err)
		return false, err
	}
	if err := view.ApplyTransaction(block, newTx, false); err != nil {
		return true, err
	}

	// validate the BVM contract
	gasOnlyTx := false
	gasStatus, err := validation.ValidateTx(newTx, block)
	if err != nil {
		if gasStatus == nil || !gasStatus.GasVaild {
			c.txPool.AddErrCache(&newTx.ID, err)
			return false, err
		}
		gasOnlyTx = true
	}

	_, err = c.txPool.AddTransaction(tx, gasOnlyTx, block.BlockHeader.Height, gasStatus.BTMValue)
	return false, err
}
