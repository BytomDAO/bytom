package database

import (
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

func newCache(fillBlockHeader fillBlockHeaderFn, fillBlockTxs fillBlockTransactionsFn, fillBlockHashes fillBlockHashesFn) cache {
	return cache{
		lruBlockHeaders: common.NewCache(maxCachedBlockHeaders),
		lruBlockTxs:     common.NewCache(maxCachedBlockTransactions),
		lruBlockHashes:  common.NewCache(maxCachedBlockHashes),

		fillBlockHeaderFn:      fillBlockHeader,
		fillBlockTransactionFn: fillBlockTxs,
		fillBlockHashesFn:      fillBlockHashes,

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
