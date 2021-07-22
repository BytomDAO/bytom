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
	for i := 0; i < maxCachedBlockHeaders+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()

		blocks[block.Hash()] = block
		blockIndexHashes[block.Height] = append(blockIndexHashes[block.Height], &hash)
	}

	fillBlockHeaderFn := func(hash *bc.Hash) (*types.BlockHeader, error) {
		return &blocks[*hash].BlockHeader, nil
	}

	fillBlockTxsFn := func(hash *bc.Hash) ([]*types.Tx, error) {
		return blocks[*hash].Transactions, nil
	}

	fillBlockHashesFn := func(height uint64) ([]*bc.Hash, error) {
		return blockIndexHashes[height], nil
	}

	fillMainChainHashFn := func(height uint64) (*bc.Hash, error) {
		return blockIndexHashes[height][0], nil
	}

	cache := newCache(fillBlockHeaderFn, fillBlockTxsFn, fillBlockHashesFn, fillMainChainHashFn, nil)

	for i := 0; i < maxCachedBlockHeaders+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		cache.lookupBlockHeader(&hash)
	}
}
