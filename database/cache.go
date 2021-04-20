package database

import (
	"fmt"
	"sync"

	"github.com/golang/groupcache/lru"
	"github.com/golang/groupcache/singleflight"

	"github.com/bytom/bytom/common"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

const (
	maxCachedBlocks       = 30
	maxCachedBlockHeaders = 4096
	maxCachedBlockHashes  = 8192
)

type fillBlockHashesFn func(height uint64) ([]*bc.Hash, error)
type fillFn func(hash *bc.Hash) (*types.Block, error)

func newCache(fill fillFn, fillBlockHashes fillBlockHashesFn) blockCache {
	return blockCache{
		lruBlockHeaders: common.NewCache(maxCachedBlockHeaders),
		lruBlockHashes:  common.NewCache(maxCachedBlockHashes),

		fillBlockHashesFn: fillBlockHashes,
		fillFn:            fill,

		lru: lru.New(maxCachedBlocks),
	}
}

type blockCache struct {
	lruBlockHeaders *common.Cache
	lruBlockHashes  *common.Cache

	fillBlockHashesFn func(uint64) ([]*bc.Hash, error)

	sf singleflight.Group

	mu     sync.Mutex
	lru    *lru.Cache
	fillFn func(hash *bc.Hash) (*types.Block, error)
}

func (c *blockCache) lookup(hash *bc.Hash) (*types.Block, error) {
	if b, ok := c.get(hash); ok {
		return b, nil
	}

	block, err := c.sf.Do(hash.String(), func() (interface{}, error) {
		b, err := c.fillFn(hash)
		if err != nil {
			return nil, err
		}

		if b == nil {
			return nil, fmt.Errorf("There are no block with given hash %s", hash.String())
		}

		c.add(b)
		return b, nil
	})
	if err != nil {
		return nil, err
	}
	return block.(*types.Block), nil
}

func (c *blockCache) get(hash *bc.Hash) (*types.Block, bool) {
	c.mu.Lock()
	block, ok := c.lru.Get(*hash)
	c.mu.Unlock()
	if block == nil {
		return nil, ok
	}
	return block.(*types.Block), ok
}

func (c *blockCache) add(block *types.Block) {
	c.mu.Lock()
	c.lru.Add(block.Hash(), block)
	c.mu.Unlock()
}
