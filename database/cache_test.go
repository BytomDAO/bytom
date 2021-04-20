package database

import (
	"testing"

	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

func TestBlockCache(t *testing.T) {
	newBlock := func(h uint64) *types.Block {
		return &types.Block{
			BlockHeader: types.BlockHeader{
				Height: h,
			},
		}
	}
	blocks := make(map[bc.Hash]*types.Block)
	blockIndexHashes := make(map[uint64][]*bc.Hash)
	for i := 0; i < maxCachedBlocks + 10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		blocks[block.Hash()] = block
		blockIndexHashes[block.Height] = append(blockIndexHashes[block.Height], &hash)
	}

	fillFn := func(hash *bc.Hash) (*types.Block, error) {
		return blocks[*hash], nil
	}

	fillBlockHashesFn := func(height uint64) ([]*bc.Hash, error) {
		return blockIndexHashes[height], nil
	}

	cache := newCache(fillFn, fillBlockHashesFn)

	for i := 0; i < maxCachedBlocks + 10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		cache.lookup(&hash)
	}

	for i := 0; i < 10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if b, _ := cache.get(&hash); b != nil {
			t.Fatalf("find old block")
		}
	}

	for i := 10; i < maxCachedBlocks + 10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if b, _ := cache.get(&hash); b == nil {
			t.Fatalf("can't find new block")
		}
	}
}
