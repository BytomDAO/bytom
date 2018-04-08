package test

import (
	"fmt"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
)

func appendBlocks(chain *protocol.Chain, num uint64) error {
	for i := uint64(0); i < num; i++ {
		block, err := NewBlock(chain, nil, []byte{byte(vm.OP_TRUE)})
		if err != nil {
			return err
		}
		if err := SolveAndUpdate(chain, block); err != nil {
			return err
		}
	}
	return nil
}

func declChain(name string, baseChain *protocol.Chain, baseHeight uint64, height uint64) (*protocol.Chain, error) {
	chainDB := dbm.NewDB(name, "leveldb", name)
	chain, err := MockChain(chainDB)
	if err != nil {
		return nil, err
	}

	if baseChain == nil {
		if err := appendBlocks(chain, height); err != nil {
			return nil, err
		}
		return chain, nil
	}

	for i := uint64(1); i <= baseHeight; i++ {
		block, err := baseChain.GetBlockByHeight(i)
		if err != nil {
			return nil, err
		}
		if err := chain.SaveBlock(block); err != nil {
			return nil, err
		}
		if err := chain.ConnectBlock(block); err != nil {
			return nil, err
		}
	}

	err = appendBlocks(chain, height-baseHeight)
	return chain, err
}

func ancestorOf(c1 *protocol.Chain, c2 *protocol.Chain) (*types.Block, error) {
	start := c1.Height()
	if c2.Height() < c1.Height() {
		start = c2.Height()
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
	if c1.Height() == c2.Height() && *c1.BestBlockHash() == *c2.BestBlockHash() {
		return nil
	}

	ancestor, err := ancestorOf(c1, c2)
	if err != nil {
		return err
	}

	processBlocks := func(dest *protocol.Chain, src *protocol.Chain, height uint64) error {
		for h := src.Height(); h > height; h-- {
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
