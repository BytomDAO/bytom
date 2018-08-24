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

	prevOrphans map[bc.Hash]*types.Block
}

func NewChain() *Chain {
	return &Chain{
		heightMap:   map[uint64]*types.Block{},
		blockMap:    map[bc.Hash]*types.Block{},
		prevOrphans: make(map[bc.Hash]*types.Block),
	}
}

func (c *Chain) BestBlockHeader() *types.BlockHeader {
	return c.bestBlockHeader
}

func (c *Chain) BestBlockHeight() uint64 {
	return c.bestBlockHeader.Height
}

func (c *Chain) CalcNextSeed(hash *bc.Hash) (*bc.Hash, error) {
	return &bc.Hash{V0: hash.V1, V1: hash.V2, V2: hash.V3, V3: hash.V0}, nil
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

func (c *Chain) GetTransactionStatus(hash *bc.Hash) (*bc.TransactionStatus, error) {
	return nil, nil
}

func (c *Chain) InMainChain(hash bc.Hash) bool {
	block, ok := c.blockMap[hash]
	if !ok {
		return false
	}
	return c.heightMap[block.Height] == block
}

func (c *Chain) ProcessBlock(block *types.Block) (bool, error) {
	if c.bestBlockHeader.Hash() == block.PreviousBlockHash {
		c.heightMap[block.Height] = block
		c.blockMap[block.Hash()] = block
		c.bestBlockHeader = &block.BlockHeader
		return false, nil
	}

	if _, ok := c.blockMap[block.PreviousBlockHash]; !ok {
		c.prevOrphans[block.PreviousBlockHash] = block
		return true, nil
	}

	c.blockMap[block.Hash()] = block
	for c.prevOrphans[block.Hash()] != nil {
		block = c.prevOrphans[block.Hash()]
		c.blockMap[block.Hash()] = block
	}

	if block.Height < c.bestBlockHeader.Height {
		return false, nil
	}

	c.bestBlockHeader = &block.BlockHeader
	for !c.InMainChain(block.Hash()) {
		c.heightMap[block.Height] = block
		block = c.blockMap[block.PreviousBlockHash]
	}
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
