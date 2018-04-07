package protocol

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/bytom/consensus"
	"github.com/bytom/database/storage"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
)

// maxCachedValidatedTxs is the max number of validated txs to cache.
const maxCachedValidatedTxs = 1000

var (
	// ErrTheDistantFuture is returned when waiting for a blockheight
	// too far in excess of the tip of the blockchain.
	ErrTheDistantFuture = errors.New("block height too far in future")
)

// Chain provides a complete, minimal blockchain database. It
// delegates the underlying storage to other objects, and uses
// validation logic from package validation to decide what
// objects can be safely stored.
type Chain struct {
	InitialBlockHash  bc.Hash
	MaxIssuanceWindow time.Duration // only used by generators

	index        *BlockIndex
	orphanManage *OrphanManage
	txPool       *TxPool

	state struct {
		cond    sync.Cond
		block   *types.Block
		hash    *bc.Hash
		height  uint64
		workSum big.Int
	}

	store Store
}

// NewChain returns a new Chain using store as the underlying storage.
func NewChain(initialBlockHash bc.Hash, store Store, txPool *TxPool) (*Chain, error) {
	c := &Chain{
		InitialBlockHash: initialBlockHash,
		orphanManage:     NewOrphanManage(),
		store:            store,
		txPool:           txPool,
	}
	c.state.cond.L = new(sync.Mutex)
	storeStatus := store.GetStoreStatus()
	c.state.hash = storeStatus.Hash
	var err error

	if c.index, err = store.LoadBlockIndex(); err != nil {
		return nil, err
	}
	bestNode := c.index.LookupNode(c.state.hash)
	c.index.SetTip(bestNode)
	if c.state.block, err = store.GetBlock(storeStatus.Hash); err != nil {
		return nil, err
	}
	return c, nil
}

// Height returns the current height of the blockchain.
func (c *Chain) Height() uint64 {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	if c.state.block == nil {
		return 0
	}
	return c.state.block.Height
}

// BestBlockHash return the hash of the chain tail block
func (c *Chain) BestBlockHash() *bc.Hash {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	return c.state.hash
}

// InMainChain checks wheather a block is in the main chain
func (c *Chain) InMainChain(hash bc.Hash) bool {
	return c.index.InMainchain(hash)
}

// Timestamp returns the latest known block timestamp.
func (c *Chain) Timestamp() uint64 {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	if c.state.block == nil {
		return 0
	}
	return c.state.block.Timestamp
}

// BestBlock returns the chain tail block
func (c *Chain) BestBlock() *types.Block {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	return c.state.block
}

// GetUtxo try to find the utxo status in db
func (c *Chain) GetUtxo(hash *bc.Hash) (*storage.UtxoEntry, error) {
	return c.store.GetUtxo(hash)
}

// GetSeed return the seed for the given block
func (c *Chain) GetSeed(height uint64, preBlock *bc.Hash) (*bc.Hash, error) {
	if height == 0 {
		return consensus.InitialSeed, nil
	} else if height%consensus.SeedPerRetarget == 0 {
		return preBlock, nil
	}

	node := c.index.LookupNode(preBlock)
	if node == nil {
		return nil, errors.New("can't find preblock in the blockindex")
	}
	return node.seed, nil
}

// GetTransactionStatus return the transaction status of give block
func (c *Chain) GetTransactionStatus(hash *bc.Hash) (*bc.TransactionStatus, error) {
	return c.store.GetTransactionStatus(hash)
}

// GetTransactionsUtxo return all the utxos that related to the txs' inputs
func (c *Chain) GetTransactionsUtxo(view *state.UtxoViewpoint, txs []*bc.Tx) error {
	return c.store.GetTransactionsUtxo(view, txs)
}

// This function must be called with mu lock in above level
func (c *Chain) setState(block *types.Block, view *state.UtxoViewpoint) error {
	blockHash := block.Hash()
	c.state.block = block
	c.state.hash = &blockHash

	if err := c.store.SaveChainStatus(block, view); err != nil {
		return err
	}

	c.state.cond.Broadcast()
	return nil
}

// BlockSoonWaiter returns a channel that
// waits for the block at the given height,
// but it is an error to wait for a block far in the future.
// WaitForBlockSoon will timeout if the context times out.
// To wait unconditionally, the caller should use WaitForBlock.
func (c *Chain) BlockSoonWaiter(ctx context.Context, height uint64) <-chan error {
	ch := make(chan error, 1)

	go func() {
		const slop = 3
		if height > c.Height()+slop {
			ch <- ErrTheDistantFuture
			return
		}

		select {
		case <-c.BlockWaiter(height):
			ch <- nil
		case <-ctx.Done():
			ch <- ctx.Err()
		}
	}()

	return ch
}

// BlockWaiter returns a channel that
// waits for the block at the given height.
func (c *Chain) BlockWaiter(height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		c.state.cond.L.Lock()
		defer c.state.cond.L.Unlock()
		for c.state.block.Height < height {
			c.state.cond.Wait()
		}
		ch <- struct{}{}
	}()

	return ch
}
