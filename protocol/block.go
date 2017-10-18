package protocol

import (
	"context"
	"time"

	"github.com/bytom/errors"
	"github.com/bytom/log"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
	"github.com/bytom/protocol/validation"
)

// maxBlockTxs limits the number of transactions
// included in each block.
const maxBlockTxs = 10000

// saveSnapshotFrequency stores how often to save a state
// snapshot to the Store.
const saveSnapshotFrequency = time.Hour

var (
	// ErrBadBlock is returned when a block is invalid.
	ErrBadBlock = errors.New("invalid block")

	// ErrStaleState is returned when the Chain does not have a current
	// blockchain state.
	ErrStaleState = errors.New("stale blockchain state")

	// ErrBadStateRoot is returned when the computed assets merkle root
	// disagrees with the one declared in a block header.
	ErrBadStateRoot = errors.New("invalid state merkle root")
)

// GetBlock returns the block at the given height, if there is one,
// otherwise it returns an error.
func (c *Chain) GetBlock(height uint64) (*legacy.Block, error) {
	return c.store.GetBlock(height)
}

// ValidateBlock validates an incoming block in advance of applying it
// to a snapshot (with ApplyValidBlock) and committing it to the
// blockchain (with CommitAppliedBlock).
func (c *Chain) ValidateBlock(block, prev *legacy.Block) error {
	blockEnts := legacy.MapBlock(block)
	prevEnts := legacy.MapBlock(prev)
	err := validation.ValidateBlock(blockEnts, prevEnts)
	if err != nil {
		return errors.Sub(ErrBadBlock, err)
	}
	return errors.Sub(ErrBadBlock, err)
}

// ApplyValidBlock creates an updated snapshot without validating the
// block.
func (c *Chain) ApplyValidBlock(block *legacy.Block) (*state.Snapshot, error) {
	newSnapshot := state.Copy(c.state.snapshot)
	err := newSnapshot.ApplyBlock(legacy.MapBlock(block))
	if err != nil {
		return nil, err
	}
	//fmt.Printf("want %v, ger %v \n", block.BlockHeader.AssetsMerkleRoot, newSnapshot.Tree.RootHash())
	if block.AssetsMerkleRoot != newSnapshot.Tree.RootHash() {
		return nil, ErrBadStateRoot
	}
	return newSnapshot, nil
}

// CommitBlock commits a block to the blockchain. The block
// must already have been applied with ApplyValidBlock or
// ApplyNewBlock, which will have produced the new snapshot that's
// required here.
//
// This function saves the block to the store and sometimes (not more
// often than saveSnapshotFrequency) saves the state tree to the
// store. New-block callbacks (via asynchronous block-processor pins)
// are triggered.
//
// TODO(bobg): rename to CommitAppliedBlock for clarity (deferred from https://github.com/chain/chain/pull/788)
func (c *Chain) CommitAppliedBlock(ctx context.Context, block *legacy.Block, snapshot *state.Snapshot) error {
	// SaveBlock is the linearization point. Once the block is committed
	// to persistent storage, the block has been applied and everything
	// else can be derived from that block.
	err := c.store.SaveBlock(block)
	if err != nil {
		return errors.Wrap(err, "storing block")
	}
	if block.Time().After(c.lastQueuedSnapshot.Add(saveSnapshotFrequency)) {
		c.queueSnapshot(ctx, block.Height, block.Time(), snapshot)
	}

	err = c.store.FinalizeBlock(ctx, block.Height)
	if err != nil {
		return errors.Wrap(err, "finalizing block")
	}

	// c.setState will update the local blockchain state and height.
	// When c.store is a txdb.Store, and c has been initialized with a
	// channel from txdb.ListenBlocks, then the above call to
	// c.store.FinalizeBlock will have done a postgresql NOTIFY and
	// that will wake up the goroutine in NewChain, which also calls
	// setHeight.  But duplicate calls with the same blockheight are
	// harmless; and the following call is required in the cases where
	// it's not redundant.
	c.setState(block, snapshot)

	return nil
}

func (c *Chain) AddBlock(ctx context.Context, block *legacy.Block) error {
	currentBlock, _ := c.State()
	if err := c.ValidateBlock(block, currentBlock); err != nil {
		return err
	}

	newSnap, err := c.ApplyValidBlock(block)
	if err != nil {
		return err
	}

	if err := c.CommitAppliedBlock(ctx, block, newSnap); err != nil {
		return err
	}

	for _, tx := range block.Transactions {
		c.txPool.RemoveTransaction(&tx.Tx.ID)
	}
	return nil
}

func (c *Chain) queueSnapshot(ctx context.Context, height uint64, timestamp time.Time, s *state.Snapshot) {
	// Non-blockingly queue the snapshot for storage.
	ps := pendingSnapshot{height: height, snapshot: s}
	select {
	case c.pendingSnapshots <- ps:
		c.lastQueuedSnapshot = timestamp
	default:
		// Skip it; saving snapshots is taking longer than the snapshotting period.
		log.Printf(ctx, "snapshot storage is taking too long; last queued at %s",
			c.lastQueuedSnapshot)
	}
}

func (c *Chain) setHeight(h uint64) {
	// We call setHeight from two places independently:
	// CommitBlock and the Postgres LISTEN goroutine.
	// This means we can get here twice for each block,
	// and any of them might be arbitrarily delayed,
	// which means h might be from the past.
	// Detect and discard these duplicate calls.

	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()

	if h <= c.state.height {
		return
	}
	c.state.height = h
	c.state.cond.Broadcast()
}
