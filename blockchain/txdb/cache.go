package txdb

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/bytom/protocol/bc/legacy"

	"github.com/golang/groupcache/lru"
	"github.com/golang/groupcache/singleflight"
)

const maxCachedBlocks = 30

func newBlockCache(fillFn func(height uint64) *legacy.Block) blockCache {
	return blockCache{
		lru:    lru.New(maxCachedBlocks),
		fillFn: fillFn,
	}
}

type blockCache struct {
	mu  sync.Mutex
	lru *lru.Cache

	fillFn func(height uint64) *legacy.Block

	single singleflight.Group // for cache misses
}

func (c *blockCache) lookup(height uint64) (*legacy.Block, error) {
	b, ok := c.get(height)
	if ok {
		return b, nil
	}

	// Cache miss; fill the block
	heightStr := strconv.FormatUint(height, 16)
	block, err := c.single.Do(heightStr, func() (interface{}, error) {
		b := c.fillFn(height)
		if b == nil {
			return nil, errors.New(fmt.Sprintf("There are no block with block height is %v", height))
		}

		c.add(b)
		return b, nil
	})
	if err != nil {
		return nil, err
	}
	return block.(*legacy.Block), nil
}

func (c *blockCache) get(height uint64) (*legacy.Block, bool) {
	c.mu.Lock()
	block, ok := c.lru.Get(height)
	c.mu.Unlock()
	if block == nil {
		return nil, ok
	}
	return block.(*legacy.Block), ok
}

func (c *blockCache) add(block *legacy.Block) {
	c.mu.Lock()
	c.lru.Add(block.Height, block)
	c.mu.Unlock()
}
