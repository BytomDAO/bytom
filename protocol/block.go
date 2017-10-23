package protocol

import (
	"time"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
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
func (c *Chain) GetBlock(hash *bc.Hash) (*legacy.Block, error) {
	return c.store.GetBlock(hash)
}

func (c *Chain) GetBlockByHeight(height uint64) (*legacy.Block, error) {
	hash, ok := c.state.mainChain[height]
	if !ok {
		return nil, nil
	}
	return c.GetBlock(hash)
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
func (c *Chain) ConnectBlock(block *legacy.Block) error {
	newSnapshot := state.Copy(c.state.snapshot)
	if err := newSnapshot.ApplyBlock(legacy.MapBlock(block)); err != nil {
		return err
	}
	if block.AssetsMerkleRoot != newSnapshot.Tree.RootHash() {
		return ErrBadStateRoot
	}

	blockHash := block.Hash()
	if err := c.store.SaveSnapshot(newSnapshot, &blockHash); err != nil {
		return err
	}
	c.state.mainChain[block.Height] = &blockHash
	if err := c.store.SaveMainchain(c.state.mainChain, &blockHash); err != nil {
		delete(c.state.mainChain, block.Height)
		return err
	}
	c.state.snapshot = newSnapshot
	c.store.SaveStoreStatus(block.Height, &blockHash)

	for _, tx := range block.Transactions {
		c.txPool.RemoveTransaction(&tx.Tx.ID)
	}
	return nil
}

func (c *Chain) getReorganizeBlocks(block *legacy.Block) ([]*legacy.Block, []*legacy.Block) {
	attachBlocks := []*legacy.Block{}
	detachBlocks := []*legacy.Block{}

	ancestor := block
	for ancestor, ok := c.orphanManage.Get(&ancestor.PreviousBlockHash); ok; {
		if c.InMainchain(ancestor) {
			break
		}
		attachBlocks = append([]*legacy.Block{ancestor}, attachBlocks...)
	}

	for n := c.state.block; n != nil; n, _ = c.GetBlock(&n.PreviousBlockHash) {
		if n.Hash() == ancestor.Hash() {
			break
		}
		detachBlocks = append(detachBlocks, n)
	}

	return attachBlocks, detachBlocks
}

func (c *Chain) AddOrphan(block *legacy.Block) error {
	attachBlocks, detachBlocks := c.getReorganizeBlocks(block)
	newSnapshot := state.Copy(c.state.snapshot)

	for _, detachBlock := range detachBlocks {
		if err := newSnapshot.DetachBlock(legacy.MapBlock(detachBlock)); err != nil {
			return err
		}
	}

	for _, attachBlock := range attachBlocks {
		if err := newSnapshot.ApplyBlock(legacy.MapBlock(attachBlock)); err != nil {
			return err
		}
	}

	blockHash := block.Hash()
	if err := c.store.SaveSnapshot(newSnapshot, &blockHash); err != nil {
		return err
	}
	for _, attachBlock := range attachBlocks {
		attachBlockHash := attachBlock.Hash()
		c.state.mainChain[attachBlock.Height] = &attachBlockHash
		c.orphanManage.Delete(&attachBlockHash)
	}
	c.state.mainChain[block.Height] = &blockHash
	if err := c.store.SaveMainchain(c.state.mainChain, &blockHash); err != nil {
		delete(c.state.mainChain, block.Height)
		return err
	}
	c.state.snapshot = newSnapshot
	c.store.SaveStoreStatus(block.Height, &blockHash)
	return nil
}

func (c *Chain) AddBlock(block *legacy.Block) (bool, error) {
	blockHash := block.Hash()
	if c.orphanManage.BlockExist(&blockHash) || c.store.BlockExist(&blockHash) {
		return c.InMainchain(block), nil
	}

	if !c.store.BlockExist(&block.PreviousBlockHash) {
		c.orphanManage.Add(block)
		return true, nil
	}

	preBlock, err := c.GetBlock(&block.PreviousBlockHash)
	if err != nil {
		return false, err
	}

	if err := c.ValidateBlock(block, preBlock); err != nil {
		return false, err
	}
	c.store.SaveBlock(block)

	if *c.state.mainChain[preBlock.Height] == block.PreviousBlockHash {
		return false, c.ConnectBlock(block)
	}

	if block.Bits > c.state.block.Bits {
		return true, c.AddOrphan(block)
	}
	return true, nil
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
