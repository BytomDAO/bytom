package protocol

import (
	"context"
	"sync"
	"time"

	"github.com/bytom/blockchain/txdb"
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
	GetSnapshot(*bc.Hash) (*state.Snapshot, error)
	GetStoreStatus() txdb.BlockStoreStateJSON

	SaveBlock(*legacy.Block) error
	SaveMainchain(map[uint64]*bc.Hash, *bc.Hash) error
	SaveSnapshot(*state.Snapshot, *bc.Hash) error
	SaveStoreStatus(uint64, *bc.Hash)
}

type OrphanManage struct {
	orphan     map[bc.Hash]*legacy.Block
	preOrphans map[bc.Hash][]*bc.Hash
	mtx        sync.RWMutex
}

func (o *OrphanManage) BlockExist(hash *bc.Hash) bool {
	o.mtx.RLock()
	_, ok := o.orphan[*hash]
	o.mtx.RUnlock()
	return ok
}

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

func (o *OrphanManage) Delete(hash *bc.Hash) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	block, ok := o.orphan[*hash]
	if !ok {
		return
	}

	delete(o.orphan, *hash)
	preOrphans, ok := o.preOrphans[block.PreviousBlockHash]
	if len(preOrphans) == 1 {
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
		cond      sync.Cond // protects height, block, snapshot
		height    uint64
		hash      *bc.Hash
		block     *legacy.Block
		snapshot  *state.Snapshot
		mainChain map[uint64]*bc.Hash
	}
	store Store

	lastQueuedSnapshot time.Time
}

type pendingSnapshot struct {
	height   uint64
	snapshot *state.Snapshot
}

// NewChain returns a new Chain using store as the underlying storage.
func NewChain(ctx context.Context, initialBlockHash bc.Hash, store Store, txPool *TxPool) (*Chain, error) {
	c := &Chain{
		InitialBlockHash: initialBlockHash,
		orphanManage: &OrphanManage{
			orphan:     make(map[bc.Hash]*legacy.Block),
			preOrphans: make(map[bc.Hash][]*bc.Hash),
		},
		store:  store,
		txPool: txPool,
	}
	c.state.cond.L = new(sync.Mutex)

	storeStatus := store.GetStoreStatus()
	c.state.height = storeStatus.Height

	if c.state.height < 1 {
		c.state.snapshot = state.Empty()
		c.state.mainChain = make(map[uint64]*bc.Hash)
	} else {
		//TODO: snapshot, mainChain version check
		c.state.hash = storeStatus.Hash
		c.state.block, _ = store.GetBlock(storeStatus.Hash)
		c.state.snapshot, _ = store.GetSnapshot(storeStatus.Hash)
		c.state.mainChain, _ = store.GetMainchain(storeStatus.Hash)
	}
	return c, nil
}

func (c *Chain) GetStore() *Store {
	return &(c.store)
}

// Height returns the current height of the blockchain.
func (c *Chain) Height() uint64 {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	return c.state.height
}

func (c *Chain) InMainchain(block *legacy.Block) bool {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	hash, ok := c.state.mainChain[block.Height]
	if !ok {
		return false
	}
	return *hash == block.Hash()
}

// TimestampMS returns the latest known block timestamp.
func (c *Chain) TimestampMS() uint64 {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	if c.state.block == nil {
		return 0
	}
	return c.state.block.TimestampMS
}

// State returns the most recent state available. It will not be current
// unless the current process is the leader. Callers should examine the
// returned block header's height if they need to verify the current state.
func (c *Chain) State() (*legacy.Block, *state.Snapshot) {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	return c.state.block, c.state.snapshot
}

func (c *Chain) setState(b *legacy.Block, s *state.Snapshot) {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	c.state.block = b
	c.state.snapshot = s
	if b != nil && b.Height > c.state.height {
		c.state.height = b.Height
		c.state.cond.Broadcast()
	}
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
