package testutil

import (
	"errors"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

type MockChain struct {
	bestBlockHeader *types.BlockHeader
	heightMap       map[uint64]*types.Block
}

func NewMockChain() *MockChain {
	return &MockChain{
		heightMap: map[uint64]*types.Block{},
	}
}

func (c *MockChain) BestBlockHeader() *types.BlockHeader {
	return c.bestBlockHeader
}

func (c *MockChain) BestBlockHeight() uint64 {
	return 0
}

func (c *MockChain) CalcNextSeed(*bc.Hash) (*bc.Hash, error) {
	return nil, nil
}

func (c *MockChain) GetBlockByHash(*bc.Hash) (*types.Block, error) {
	return nil, nil
}

func (c *MockChain) GetBlockByHeight(height uint64) (*types.Block, error) {
	block, ok := c.heightMap[height]
	if !ok {
		return nil, errors.New("can't find block")
	}
	return block, nil
}

func (c *MockChain) GetHeaderByHash(*bc.Hash) (*types.BlockHeader, error) {
	return nil, nil
}

func (c *MockChain) GetHeaderByHeight(height uint64) (*types.BlockHeader, error) {
	block, ok := c.heightMap[height]
	if !ok {
		return nil, errors.New("can't find block")
	}
	return &block.BlockHeader, nil
}

func (c *MockChain) InMainChain(bc.Hash) bool {
	return true
}

func (c *MockChain) ProcessBlock(*types.Block) (bool, error) {
	return false, nil
}

func (c *MockChain) SetBestBlockHeader(header *types.BlockHeader) {
	c.bestBlockHeader = header
}

func (c *MockChain) SetBlockByHeight(height uint64, block *types.Block) {
	c.heightMap[height] = block
}

func (c *MockChain) ValidateTx(*types.Tx) (bool, error) {
	return false, nil
}
