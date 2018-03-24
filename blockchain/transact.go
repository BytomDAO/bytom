package blockchain

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc/types"
)

// finalizeTxWait calls FinalizeTx and then waits for confirmation of
// the transaction.  A nil error return means the transaction is
// confirmed on the blockchain.  ErrRejected means a conflicting tx is
// on the blockchain.  context.DeadlineExceeded means ctx is an
// expiring context that timed out.
func (bcr *BlockchainReactor) finalizeTxWait(ctx context.Context, txTemplate *txbuilder.Template, waitUntil string) error {
	// Use the current generator height as the lower bound of the block height
	// that the transaction may appear in.
	localHeight := bcr.chain.Height()
	//generatorHeight := localHeight

	log.WithField("localHeight", localHeight).Info("Starting to finalize transaction")

	err := txbuilder.FinalizeTx(ctx, bcr.chain, txTemplate.Transaction)
	if err != nil {
		return err
	}
	if waitUntil == "none" {
		return nil
	}

	//TODO:complete finalizeTxWait
	//height, err := a.waitForTxInBlock(ctx, txTemplate.Transaction, generatorHeight)
	if err != nil {
		return err
	}
	if waitUntil == "confirmed" {
		return nil
	}

	return nil
}

func (bcr *BlockchainReactor) waitForTxInBlock(ctx context.Context, tx *types.Tx, height uint64) (uint64, error) {
	log.Printf("waitForTxInBlock function")
	for {
		height++
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		case <-bcr.chain.BlockWaiter(height):
			b, err := bcr.chain.GetBlockByHeight(height)
			if err != nil {
				return 0, errors.Wrap(err, "getting block that just landed")
			}
			for _, confirmed := range b.Transactions {
				if confirmed.ID == tx.ID {
					// confirmed
					return height, nil
				}
			}

			// might still be in pool or might be rejected; we can't
			// tell definitively until its max time elapses.
			// Re-insert into the pool in case it was dropped.
			err = txbuilder.FinalizeTx(ctx, bcr.chain, tx)
			if err != nil {
				return 0, err
			}

			// TODO(jackson): Do simple rejection checks like checking if
			// the tx's blockchain prevouts still exist in the state tree.
		}
	}
}
