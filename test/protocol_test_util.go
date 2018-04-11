package test

import (
	"fmt"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/types"
)

func declChain(name string, baseChain *protocol.Chain, baseHeight uint64, height uint64) (*protocol.Chain, error) {
	chainDB := dbm.NewDB(name, "leveldb", name)
	chain, _, _, err := MockChain(chainDB)
	if err != nil {
		return nil, err
	}

	if baseChain == nil {
		if err := AppendBlocks(chain, height); err != nil {
			return nil, err
		}
		return chain, nil
	}

	for i := uint64(1); i <= baseHeight; i++ {
		block, err := baseChain.GetBlockByHeight(i)
		if err != nil {
			return nil, err
		}
		if err := SolveAndUpdate(chain, block); err != nil {
			return nil, err
		}
	}

	err = AppendBlocks(chain, height-baseHeight)
	return chain, err
}

func ancestorOf(c1 *protocol.Chain, c2 *protocol.Chain) (*types.Block, error) {
	start := c1.BestBlockHeight()
	if c2.BestBlockHeight() < c1.BestBlockHeight() {
		start = c2.BestBlockHeight()
	}

	for i := start; i >= 0; i-- {
		b1, err := c1.GetBlockByHeight(i)
		if err != nil {
			return nil, err
		}
		b2, err := c2.GetBlockByHeight(i)
		if err != nil {
			return nil, err
		}
		if b1.Hash() == b2.Hash() {
			return b1, nil
		}
	}
	return nil, fmt.Errorf("can't find ancestor")
}

func merge(c1 *protocol.Chain, c2 *protocol.Chain) error {
	// c1 and c2 are same
	if c1.BestBlockHeight() == c2.BestBlockHeight() && *c1.BestBlockHash() == *c2.BestBlockHash() {
		return nil
	}

	ancestor, err := ancestorOf(c1, c2)
	if err != nil {
		return err
	}

	processBlocks := func(dest *protocol.Chain, src *protocol.Chain, height uint64) error {
		for h := src.BestBlockHeight(); h > height; h-- {
			block, err := src.GetBlockByHeight(h)
			if err != nil {
				return err
			}
			_, err = dest.ProcessBlock(block)
			if err != nil {
				return err
			}
		}
		return nil
	}

	if err := processBlocks(c1, c2, ancestor.Height); err != nil {
		return err
	}
	return processBlocks(c2, c1, ancestor.Height)
}
