package protocol

import (
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
	"github.com/bytom/protocol/validation"
)

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

func (c *Chain) BlockExist(hash *bc.Hash) bool {
	return c.orphanManage.BlockExist(hash) || c.store.BlockExist(hash)
}

func (c *Chain) GetBlockByHash(hash *bc.Hash) (*legacy.Block, error) {
	return c.store.GetBlock(hash)
}

func (c *Chain) GetBlockByHeight(height uint64) (*legacy.Block, error) {
	c.state.cond.L.Lock()
	hash, ok := c.state.mainChain[height]
	c.state.cond.L.Unlock()
	if !ok {
		return nil, nil
	}
	return c.GetBlockByHash(hash)
}

// ValidateBlock validates an incoming block in advance of applying it
// to a snapshot (with ApplyValidBlock) and committing it to the
// blockchain (with CommitAppliedBlock).
func (c *Chain) ValidateBlock(block, prev *legacy.Block) error {
	blockEnts := legacy.MapBlock(block)
	prevEnts := legacy.MapBlock(prev)
	if err := validation.ValidateBlock(blockEnts, prevEnts); err != nil {
		return errors.Sub(ErrBadBlock, err)
	}
	return nil
}

// ApplyValidBlock creates an updated snapshot without validating the
// block.
func (c *Chain) connectBlock(block *legacy.Block) error {
	newSnapshot := state.Copy(c.state.snapshot)
	if err := newSnapshot.ApplyBlock(legacy.MapBlock(block)); err != nil {
		return err
	}

	blockHash := block.Hash()
	if err := c.setState(block, newSnapshot, map[uint64]*bc.Hash{block.Height: &blockHash}); err != nil {
		return err
	}

	for _, tx := range block.Transactions {
		c.txPool.RemoveTransaction(&tx.Tx.ID)
	}
	return nil
}

func (c *Chain) getReorganizeBlocks(block *legacy.Block) ([]*legacy.Block, []*legacy.Block) {
	attachBlocks := []*legacy.Block{}
	detachBlocks := []*legacy.Block{}
	ancestor := block

	for !c.InMainchain(ancestor) {
		attachBlocks = append([]*legacy.Block{ancestor}, attachBlocks...)
		ancestor, _ = c.GetBlockByHash(&ancestor.PreviousBlockHash)
	}

	for d := c.state.block; d.Hash() != ancestor.Hash(); d, _ = c.GetBlockByHash(&d.PreviousBlockHash) {
		detachBlocks = append(detachBlocks, d)
	}

	return attachBlocks, detachBlocks
}

func (c *Chain) reorganizeChain(block *legacy.Block) error {
	attachBlocks, detachBlocks := c.getReorganizeBlocks(block)
	newSnapshot := state.Copy(c.state.snapshot)
	chainChanges := map[uint64]*bc.Hash{}

	for _, d := range detachBlocks {
		if err := newSnapshot.DetachBlock(legacy.MapBlock(d)); err != nil {
			return err
		}
	}

	for _, a := range attachBlocks {
		if err := newSnapshot.ApplyBlock(legacy.MapBlock(a)); err != nil {
			return err
		}
		aHash := a.Hash()
		chainChanges[a.Height] = &aHash
	}

	return c.setState(block, newSnapshot, chainChanges)
}

func (c *Chain) SaveBlock(block *legacy.Block) error {
	preBlock, err := c.GetBlockByHash(&block.PreviousBlockHash)
	if err != nil {
		return err
	}
	if err := c.ValidateBlock(block, preBlock); err != nil {
		return err
	}
	c.store.SaveBlock(block)

	preorphans, ok := c.orphanManage.preOrphans[block.Hash()]
	if !ok {
		return nil
	}
	for _, preorphan := range preorphans {
		orphanBlock, ok := c.orphanManage.Get(preorphan)
		if !ok {
			continue
		}
		c.SaveBlock(orphanBlock)
		c.orphanManage.Delete(preorphan)
	}
	return nil
}

func (c *Chain) ProcessBlock(block *legacy.Block) (bool, error) {
	if blockHash := block.Hash(); c.BlockExist(&blockHash) {
		return false, nil
	}
	if !c.BlockExist(&block.PreviousBlockHash) {
		c.orphanManage.Add(block)
		return true, nil
	}
	if err := c.SaveBlock(block); err != nil {
		return false, err
	}

	c.state.cond.L.Lock()
	if c.state.block.Hash() == block.PreviousBlockHash {
		defer c.state.cond.L.Unlock()
		return false, c.connectBlock(block)
	}

	if block.Height > c.state.height && block.Bits >= c.state.block.Bits {
		defer c.state.cond.L.Unlock()
		return false, c.reorganizeChain(block)
	}
	c.state.cond.L.Unlock()
	return false, nil
}
