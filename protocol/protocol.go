package protocol

import (
	"context"
	"math/big"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/config"
	"github.com/bytom/database/storage"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
)

const (
	maxProcessBlockChSize = 1024
)

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
	index          *BlockIndex
	orphanManage   *OrphanManage
	txPool         *TxPool
	processBlockCh chan *processBlockMsg

	state struct {
		cond    sync.Cond
		hash    *bc.Hash
		height  uint64
		workSum *big.Int
	}

	store Store
}

// NewChain returns a new Chain using store as the underlying storage.
func NewChain(store Store, txPool *TxPool) (*Chain, error) {
	c := &Chain{
		orphanManage:   NewOrphanManage(),
		store:          store,
		txPool:         txPool,
		processBlockCh: make(chan *processBlockMsg, maxProcessBlockChSize),
	}
	c.state.cond.L = new(sync.Mutex)

	var err error
	if storeStatus := store.GetStoreStatus(); storeStatus.Hash != nil {
		c.state.hash = storeStatus.Hash
	} else {
		if err = c.initChainStatus(); err != nil {
			return nil, err
		}
	}

	if c.index, err = store.LoadBlockIndex(); err != nil {
		return nil, err
	}

	bestNode := c.index.GetNode(c.state.hash)
	c.index.SetMainChain(bestNode)
	c.state.height = bestNode.height
	c.state.workSum = bestNode.workSum
	go c.blockProcesser()
	return c, nil
}

func (c *Chain) initChainStatus() error {
	genesisBlock := config.GenerateGenesisBlock()
	txStatus := bc.NewTransactionStatus()
	for i, _ := range genesisBlock.Transactions {
		txStatus.SetStatus(i, false)
	}

	if err := c.store.SaveBlock(genesisBlock, txStatus); err != nil {
		return err
	}

	utxoView := state.NewUtxoViewpoint()
	bcBlock := types.MapBlock(genesisBlock)
	if err := utxoView.ApplyBlock(bcBlock, txStatus); err != nil {
		return err
	}

	if err := c.store.SaveChainStatus(genesisBlock, utxoView); err != nil {
		return err
	}

	hash := genesisBlock.Hash()
	c.state.hash = &hash
	return nil
}

// Height returns the current height of the blockchain.
func (c *Chain) Height() uint64 {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	return c.state.height
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

// BestBlock returns the chain tail block
func (c *Chain) BestBlockHeader() *types.BlockHeader {
	node := c.index.BestNode()
	return node.blockHeader()
}

// GetUtxo try to find the utxo status in db
func (c *Chain) GetUtxo(hash *bc.Hash) (*storage.UtxoEntry, error) {
	return c.store.GetUtxo(hash)
}

// CalcNextSeed return the seed for the given block
func (c *Chain) CalcNextSeed(preBlock *bc.Hash) (*bc.Hash, error) {
	node := c.index.GetNode(preBlock)
	if node == nil {
		return nil, errors.New("can't find preblock in the blockindex")
	}
	return node.CalcNextSeed(), nil
}

// CalcNextBits return the seed for the given block
func (c *Chain) CalcNextBits(preBlock *bc.Hash) (uint64, error) {
	node := c.index.GetNode(preBlock)
	if node == nil {
		return 0, errors.New("can't find preblock in the blockindex")
	}
	return node.CalcNextBits(), nil
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
	if err := c.store.SaveChainStatus(block, view); err != nil {
		return err
	}

	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()

	blockHash := block.Hash()
	node := c.index.GetNode(&blockHash)
	c.index.SetMainChain(node)
	c.state.hash = &blockHash
	c.state.height = node.height
	c.state.workSum = node.workSum

	log.WithFields(log.Fields{
		"height":  c.state.height,
		"hash":    c.state.hash.String(),
		"workSum": c.state.workSum,
	}).Debug("Chain best status has been changed")
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
		for c.state.height < height {
			c.state.cond.Wait()
		}
		ch <- struct{}{}
	}()

	return ch
}
