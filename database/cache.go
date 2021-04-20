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
	maxCachedBlocks            = 30
	maxCachedBlockHeaders      = 4096
	maxCachedBlockTransactions = 1024
	maxCachedBlockHashes       = 8192
)

type fillBlockHeaderFn func(hash *bc.Hash) (*types.BlockHeader, error)
type fillBlockTransactionsFn func(hash *bc.Hash) ([]*types.Tx, error)
type fillBlockHashesFn func(height uint64) ([]*bc.Hash, error)
type fillFn func(hash *bc.Hash) (*types.Block, error)

func newCache(fillBlockHeader fillBlockHeaderFn, fillBlockTxs fillBlockTransactionsFn, fill fillFn, fillBlockHashes fillBlockHashesFn) cache {
	return cache{
		lruBlockHeaders: common.NewCache(maxCachedBlockHeaders),
		lruBlockTxs:     common.NewCache(maxCachedBlockTransactions),
		lruBlockHashes:  common.NewCache(maxCachedBlockHashes),

		fillBlockHeaderFn:      fillBlockHeader,
		fillBlockTransactionFn: fillBlockTxs,
		fillBlockHashesFn:      fillBlockHashes,
		fillFn:                 fill,

		lru: lru.New(maxCachedBlocks),
	}
}

type cache struct {
	lruBlockHeaders *common.Cache
	lruBlockTxs     *common.Cache
	lruBlockHashes  *common.Cache

	fillBlockHashesFn      func(uint64) ([]*bc.Hash, error)
	fillBlockTransactionFn func(hash *bc.Hash) ([]*types.Tx, error)
	fillBlockHeaderFn      func(hash *bc.Hash) (*types.BlockHeader, error)
	fillFn                 func(hash *bc.Hash) (*types.Block, error)

	sf singleflight.Group

	mu  sync.Mutex
	lru *lru.Cache
}

func (c *cache) lookup(hash *bc.Hash) (*types.Block, error) {
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

func (c *cache) get(hash *bc.Hash) (*types.Block, bool) {
	c.mu.Lock()
	block, ok := c.lru.Get(*hash)
	c.mu.Unlock()
	if block == nil {
		return nil, ok
	}
	return block.(*types.Block), ok
}

func (c *cache) add(block *types.Block) {
	c.mu.Lock()
	c.lru.Add(block.Hash(), block)
	c.mu.Unlock()
}
