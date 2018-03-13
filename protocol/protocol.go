package protocol

import (
	"context"
	"sync"
	"time"

	"github.com/bytom/blockchain/txdb"
	"github.com/bytom/blockchain/txdb/storage"
	"github.com/bytom/consensus"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
)

// maxCachedValidatedTxs is the max number of validated txs to cache.
const maxCachedValidatedTxs = 1000

var (
	// ErrTheDistantFuture is returned when waiting for a blockheight
	// too far in excess of the tip of the blockchain.
	ErrTheDistantFuture = errors.New("block height too far in future")
)

// Store provides storage for blockchain data: blocks and state tree
// snapshots.
//
// Note, this is different from a state snapshot. A state snapshot
// provides access to the state at a given point in time -- outputs
// and issuance memory. The Chain type uses Store to load state
// from storage and persist validated data.
type Store interface {
	BlockExist(*bc.Hash) bool

	GetBlock(*bc.Hash) (*legacy.Block, error)
	GetMainchain(*bc.Hash) (map[uint64]*bc.Hash, error)
	GetStoreStatus() txdb.BlockStoreStateJSON
	GetSeed(*bc.Hash) (*bc.Hash, error)
	GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error)
	GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error
	GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)

	SaveBlock(*legacy.Block, *bc.TransactionStatus, *bc.Hash) error
	SaveChainStatus(*legacy.Block, *state.UtxoViewpoint, map[uint64]*bc.Hash) error
}

// OrphanManage is use to handle all the orphan block
type OrphanManage struct {
	//TODO: add orphan cached block limit
	orphan     map[bc.Hash]*legacy.Block
	preOrphans map[bc.Hash][]*bc.Hash
	mtx        sync.RWMutex
}

// NewOrphanManage return a new orphan block
func NewOrphanManage() *OrphanManage {
	return &OrphanManage{
		orphan:     make(map[bc.Hash]*legacy.Block),
		preOrphans: make(map[bc.Hash][]*bc.Hash),
	}
}

// BlockExist check is the block in OrphanManage
func (o *OrphanManage) BlockExist(hash *bc.Hash) bool {
	o.mtx.RLock()
	_, ok := o.orphan[*hash]
	o.mtx.RUnlock()
	return ok
}

// Add will add the block to OrphanManage
func (o *OrphanManage) Add(block *legacy.Block) {
	blockHash := block.Hash()
	o.mtx.Lock()
	defer o.mtx.Unlock()

	if _, ok := o.orphan[blockHash]; ok {
		return
	}

	o.orphan[blockHash] = block
	o.preOrphans[block.PreviousBlockHash] = append(o.preOrphans[block.PreviousBlockHash], &blockHash)
}

// Delete will delelte the block from OrphanManage
func (o *OrphanManage) Delete(hash *bc.Hash) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	block, ok := o.orphan[*hash]
	if !ok {
		return
	}
	delete(o.orphan, *hash)

	preOrphans, ok := o.preOrphans[block.PreviousBlockHash]
	if !ok || len(preOrphans) == 1 {
		delete(o.preOrphans, block.PreviousBlockHash)
		return
	}

	for i, preOrphan := range preOrphans {
		if preOrphan == hash {
			o.preOrphans[block.PreviousBlockHash] = append(preOrphans[:i], preOrphans[i+1:]...)
			return
		}
	}
}

// Get return the orphan block by hash
func (o *OrphanManage) Get(hash *bc.Hash) (*legacy.Block, bool) {
	o.mtx.RLock()
	block, ok := o.orphan[*hash]
	o.mtx.RUnlock()
	return block, ok
}

// Chain provides a complete, minimal blockchain database. It
// delegates the underlying storage to other objects, and uses
// validation logic from package validation to decide what
// objects can be safely stored.
type Chain struct {
	InitialBlockHash  bc.Hash
	MaxIssuanceWindow time.Duration // only used by generators

	orphanManage *OrphanManage
	txPool       *TxPool

	state struct {
		cond      sync.Cond
		block     *legacy.Block
		hash      *bc.Hash
		mainChain map[uint64]*bc.Hash
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

	if storeStatus.Hash == nil {
		c.state.mainChain = make(map[uint64]*bc.Hash)
		return c, nil
	}

	c.state.hash = storeStatus.Hash
	var err error
	if c.state.block, err = store.GetBlock(storeStatus.Hash); err != nil {
		return nil, err
	}
	if c.state.mainChain, err = store.GetMainchain(storeStatus.Hash); err != nil {
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

func (c *Chain) inMainchain(block *legacy.Block) bool {
	hash, ok := c.state.mainChain[block.Height]
	if !ok {
		return false
	}
	return *hash == block.Hash()
}

// InMainChain checks wheather a block is in the main chain
func (c *Chain) InMainChain(height uint64, hash bc.Hash) bool {
	c.state.cond.L.Lock()
	h, ok := c.state.mainChain[height]
	c.state.cond.L.Unlock()
	if !ok {
		return false
	}

	return *h == hash
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
func (c *Chain) BestBlock() *legacy.Block {
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
	return c.store.GetSeed(preBlock)
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
func (c *Chain) setState(block *legacy.Block, view *state.UtxoViewpoint, m map[uint64]*bc.Hash) error {
	blockHash := block.Hash()
	c.state.block = block
	c.state.hash = &blockHash
	for k, v := range m {
		c.state.mainChain[k] = v
	}

	if err := c.store.SaveChainStatus(block, view, c.state.mainChain); err != nil {
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
