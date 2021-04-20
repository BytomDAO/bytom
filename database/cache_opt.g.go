package database

import "github.com/bytom/bytom/protocol/bc/types"

func (c *blockCache) removeBlockHeader(blockHeader *types.BlockHeader) {
	c.lruBlockHeaders.Remove(blockHeader.Hash())
}
