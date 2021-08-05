package database

import (
	"encoding/hex"
	"strconv"

	"github.com/golang/groupcache/singleflight"

	"github.com/bytom/bytom/common"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

const (
	maxCachedBlockHeaders      = 2048
	maxCachedBlockTransactions = 1024
	maxCachedBlockHashes       = 1024
	maxCachedMainChainHashes   = 1024
	maxCheckPoints             = 256
)

type fillBlockHeaderFn func(hash *bc.Hash) (*types.BlockHeader, error)
type fillBlockTransactionsFn func(hash *bc.Hash) ([]*types.Tx, error)
type fillBlockHashesFn func(height uint64) ([]*bc.Hash, error)
type fillMainChainHashFn func(height uint64) (*bc.Hash, error)
type fillCheckPointFn func(key []byte) (*state.Checkpoint, error)

func newCache(fillBlockHeader fillBlockHeaderFn, fillBlockTxs fillBlockTransactionsFn, fillBlockHashes fillBlockHashesFn, fillMainChainHash fillMainChainHashFn, fillCheckPoint fillCheckPointFn) cache {
	return cache{
		lruBlockHeaders:    common.NewCache(maxCachedBlockHeaders),
		lruBlockTxs:        common.NewCache(maxCachedBlockTransactions),
		lruBlockHashes:     common.NewCache(maxCachedBlockHashes),
		lruMainChainHashes: common.NewCache(maxCachedMainChainHashes),
		lruCheckPoints:     common.NewCache(maxCheckPoints),

		fillBlockHeaderFn:      fillBlockHeader,
		fillBlockTransactionFn: fillBlockTxs,
		fillBlockHashesFn:      fillBlockHashes,
		fillMainChainHashFn:    fillMainChainHash,
		fillCheckPointFn:       fillCheckPoint,
	}
}

type cache struct {
	lruBlockHeaders    *common.Cache
	lruBlockTxs        *common.Cache
	lruBlockHashes     *common.Cache
	lruMainChainHashes *common.Cache
	lruCheckPoints     *common.Cache

	fillBlockHashesFn      func(uint64) ([]*bc.Hash, error)
	fillBlockTransactionFn func(hash *bc.Hash) ([]*types.Tx, error)
	fillBlockHeaderFn      func(hash *bc.Hash) (*types.BlockHeader, error)
	fillMainChainHashFn    func(uint64) (*bc.Hash, error)
	fillCheckPointFn       func(key []byte) (*state.Checkpoint, error)

	sf singleflight.Group
}

func (c *cache) removeBlockHeader(blockHeader *types.BlockHeader) {
	c.lruBlockHeaders.Remove(blockHeader.Hash())
}

func (c *cache) lookupBlockHashesByHeight(height uint64) ([]*bc.Hash, error) {
	if hashes, ok := c.lruBlockHashes.Get(height); ok {
		return hashes.([]*bc.Hash), nil
	}

	heightStr := strconv.FormatUint(height, 10)
	hashes, err := c.sf.Do("BlockHashesByHeight:"+heightStr, func() (interface{}, error) {
		hashes, err := c.fillBlockHashesFn(height)
		if err != nil {
			return nil, err
		}

		c.lruBlockHashes.Add(height, hashes)
		return hashes, nil
	})
	if err != nil {
		return nil, err
	}

	return hashes.([]*bc.Hash), nil
}

func (c *cache) removeBlockHashes(height uint64) {
	c.lruBlockHashes.Remove(height)
}

func (c *cache) lookupBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	if data, ok := c.lruBlockHeaders.Get(*hash); ok {
		return data.(*types.BlockHeader), nil
	}

	blockHeader, err := c.sf.Do("BlockHeader:"+hash.String(), func() (interface{}, error) {
		blockHeader, err := c.fillBlockHeaderFn(hash)
		if err != nil {
			return nil, err
		}

		c.lruBlockHeaders.Add(blockHeader.Hash(), blockHeader)
		return blockHeader, nil
	})
	if err != nil {
		return nil, err
	}
	return blockHeader.(*types.BlockHeader), nil
}

func (c *cache) lookupBlockTxs(hash *bc.Hash) ([]*types.Tx, error) {
	if data, ok := c.lruBlockTxs.Get(*hash); ok {
		return data.([]*types.Tx), nil
	}

	blockTxs, err := c.sf.Do("BlockTxs:"+hash.String(), func() (interface{}, error) {
		blockTxs, err := c.fillBlockTransactionFn(hash)
		if err != nil {
			return nil, err
		}

		c.lruBlockTxs.Add(*hash, blockTxs)
		return blockTxs, nil
	})
	if err != nil {
		return nil, err
	}
	return blockTxs.([]*types.Tx), nil
}

func (c *cache) lookupMainChainHash(height uint64) (*bc.Hash, error) {
	if hash, ok := c.lruMainChainHashes.Get(height); ok {
		return hash.(*bc.Hash), nil
	}

	heightStr := strconv.FormatUint(height, 10)
	hash, err := c.sf.Do("BlockHashByHeight:"+heightStr, func() (interface{}, error) {
		hash, err := c.fillMainChainHashFn(height)
		if err != nil {
			return nil, err
		}

		c.lruMainChainHashes.Add(height, hash)
		return hash, nil
	})
	if err != nil {
		return nil, err
	}

	return hash.(*bc.Hash), nil
}

func (c *cache) removeMainChainHash(height uint64) {
	c.lruMainChainHashes.Remove(height)
}

func (c *cache) lookupCheckPoint(key []byte) (*state.Checkpoint, error) {
	keyStr := hex.EncodeToString(key)
	if data, ok := c.lruCheckPoints.Get(keyStr); ok {
		return data.(*state.Checkpoint), nil
	}

	checkpoint, err := c.sf.Do("CheckPoint:"+string(key), func() (interface{}, error) {
		checkPoint, err := c.fillCheckPointFn(key)
		if err != nil {
			return nil, err
		}

		c.lruCheckPoints.Add(keyStr, checkPoint)
		return checkPoint, nil
	})
	if err != nil {
		return nil, err
	}

	return checkpoint.(*state.Checkpoint), nil
}

func (c *cache) removeCheckPoint(key []byte) {
	c.lruCheckPoints.Remove(hex.EncodeToString(key))
}
