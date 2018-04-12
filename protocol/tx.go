package protocol

import (
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
	"github.com/bytom/protocol/validation"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// GetTransactionStatus return the transaction status of give block
func (c *Chain) GetTransactionStatus(hash *bc.Hash) (*bc.TransactionStatus, error) {
	return c.store.GetTransactionStatus(hash)
}

// GetTransactionsUtxo return all the utxos that related to the txs' inputs
func (c *Chain) GetTransactionsUtxo(view *state.UtxoViewpoint, txs []*bc.Tx) error {
	return c.store.GetTransactionsUtxo(view, txs)
}

// ValidateTx validates the given transaction. A cache holds
// per-transaction validation results and is consulted before
// performing full validation.
func (c *Chain) ValidateTx(tx *types.Tx) (bool, error) {
	if ok := c.txPool.HaveTransaction(&tx.ID); ok {
		return false, c.txPool.GetErrCache(&tx.ID)
	}

	view := c.txPool.GetTransactionUTXO(tx.Tx)
	if err := c.GetTransactionsUtxo(view, []*bc.Tx{tx.Tx}); err != nil {
		return true, err
	}

	bh := c.BestBlockHeader()
	block := types.MapBlock(&types.Block{BlockHeader: *bh})
	if err := view.ApplyTransaction(block, tx.Tx, false); err != nil {
		return true, err
	}

	gasStatus, err := validation.ValidateTx(tx.Tx, block)
	if gasStatus.GasVaild == false {
		c.txPool.AddErrCache(&tx.ID, err)
		return false, err
	}

	_, err = c.txPool.AddTransaction(tx, err != nil, block.BlockHeader.Height, gasStatus.BTMValue)
	return false, err
}
