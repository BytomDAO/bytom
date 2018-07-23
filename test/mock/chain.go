package mock

import (
	"errors"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

type Chain struct {
	bestBlockHeader *types.BlockHeader
	heightMap       map[uint64]*types.Block
	blockMap        map[bc.Hash]*types.Block
}

func NewChain() *Chain {
	return &Chain{
		heightMap: map[uint64]*types.Block{},
		blockMap:  map[bc.Hash]*types.Block{},
	}
}

func (c *Chain) BestBlockHeader() *types.BlockHeader {
	return c.bestBlockHeader
}

func (c *Chain) BestBlockHeight() uint64 {
	return 0
}

func (c *Chain) CalcNextSeed(*bc.Hash) (*bc.Hash, error) {
	return nil, nil
}

func (c *Chain) GetBlockByHash(hash *bc.Hash) (*types.Block, error) {
	block, ok := c.blockMap[*hash]
	if !ok {
		return nil, errors.New("can't find block")
	}
	return block, nil
}

func (c *Chain) GetBlockByHeight(height uint64) (*types.Block, error) {
	block, ok := c.heightMap[height]
	if !ok {
		return nil, errors.New("can't find block")
	}
	return block, nil
}

func (c *Chain) GetHeaderByHash(hash *bc.Hash) (*types.BlockHeader, error) {
	block, ok := c.blockMap[*hash]
	if !ok {
		return nil, errors.New("can't find block")
	}
	return &block.BlockHeader, nil
}

func (c *Chain) GetHeaderByHeight(height uint64) (*types.BlockHeader, error) {
	block, ok := c.heightMap[height]
	if !ok {
		return nil, errors.New("can't find block")
	}
	return &block.BlockHeader, nil
}

func (c *Chain) InMainChain(bc.Hash) bool { return true }

func (c *Chain) ProcessBlock(*types.Block) (bool, error) {
	return false, nil
}

func (c *Chain) SetBestBlockHeader(header *types.BlockHeader) {
	c.bestBlockHeader = header
}

func (c *Chain) SetBlockByHeight(height uint64, block *types.Block) {
	c.heightMap[height] = block
	c.blockMap[block.Hash()] = block
}

func (c *Chain) ValidateTx(*types.Tx) (bool, error) {
	return false, nil
}
