package protocol

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
	"github.com/bytom/bytom/protocol/validation"
	"github.com/bytom/bytom/protocol/vm"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

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

	if c.txPool.IsDust(tx) {
		c.txPool.AddErrCache(&tx.ID, ErrDustTx)
		return false, ErrDustTx
	}

	bh := c.BestBlockHeader()
	gasStatus, err := validation.ValidateTx(tx.Tx, types.MapBlock(&types.Block{BlockHeader: *bh}), c.ProgramConverter)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "tx_id": tx.Tx.ID.String(), "error": err}).Info("transaction status fail")
		c.txPool.AddErrCache(&tx.ID, err)
		return false, err
	}

	return c.txPool.ProcessTransaction(tx, bh.Height, gasStatus.BTMValue)
}

//ProgramConverter convert program. Only for BCRP now
func (c *Chain) ProgramConverter(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	if len(insts) != 2 {
		return nil, errors.New("unsupport program")
	}

	var hash [32]byte
	copy(hash[:], insts[1].Data)

	return c.store.GetContract(hash)
}
