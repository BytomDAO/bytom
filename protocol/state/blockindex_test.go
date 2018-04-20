package state

import (
	"math/big"
	"testing"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

func TestCalcPastMedianTime(t *testing.T) {
	cases := []struct {
		Timestamps []uint64
		MedianTime uint64
	}{
		{
			Timestamps: []uint64{1},
			MedianTime: 1,
		},
		{
			Timestamps: []uint64{1, 2},
			MedianTime: 2,
		},
		{
			Timestamps: []uint64{1, 3, 2},
			MedianTime: 2,
		},
		{
			Timestamps: []uint64{1, 3, 2, 3},
			MedianTime: 3,
		},
		{
			Timestamps: []uint64{1, 2, 3, 4, 5, 6, 7, 8, 11, 10, 9},
			MedianTime: 6,
		},
		{
			Timestamps: []uint64{1, 2, 3, 4, 5, 6, 7, 8, 11, 10, 9, 11, 11, 11, 14},
			MedianTime: 10,
		},
	}

	for idx, c := range cases {
		var parentNode *BlockNode
		for i, _ := range c.Timestamps {
			blockHeader := &types.BlockHeader{
				Height:    uint64(i),
				Timestamp: c.Timestamps[i],
			}

			blockNode, err := NewBlockNode(blockHeader, parentNode)
			if err != nil {
				t.Fatal(err)
			}
			parentNode = blockNode
		}

		medianTime := parentNode.CalcPastMedianTime()
		if medianTime != c.MedianTime {
			t.Fatalf("calc median timestamp failed, index: %d, expected: %d, have: %d", idx, c.MedianTime, medianTime)
		}
	}
}

func TestCalcNextBits(t *testing.T) {
	targetTimeSpan := uint64(consensus.BlocksPerRetarget * consensus.TargetSecondsPerBlock)
	cases := []struct {
		parentNode  *BlockNode
		currentNode *BlockNode
		bits        uint64
	}{
		{
			currentNode: &BlockNode{
				Height: 0,
				Bits:   1000,
			},
			bits: 1000,
		},
		{
			currentNode: &BlockNode{
				Height: consensus.BlocksPerRetarget - 1,
				Bits:   1000,
			},
			bits: 1000,
		},
		{
			parentNode: &BlockNode{
				Height:    0,
				Timestamp: 0,
			},
			currentNode: &BlockNode{
				Height:    consensus.BlocksPerRetarget,
				Bits:      difficulty.BigToCompact(big.NewInt(1000)),
				Timestamp: targetTimeSpan,
			},
			bits: difficulty.BigToCompact(big.NewInt(1000)),
		},
		{
			parentNode: &BlockNode{
				Height:    0,
				Timestamp: 0,
			},
			currentNode: &BlockNode{
				Height:    consensus.BlocksPerRetarget,
				Bits:      difficulty.BigToCompact(big.NewInt(1000)),
				Timestamp: targetTimeSpan * 2,
			},
			bits: difficulty.BigToCompact(big.NewInt(2000)),
		},
	}

	for i, c := range cases {
		c.currentNode.Parent = c.parentNode
		bits := c.currentNode.CalcNextBits()
		if bits != c.bits {
			t.Fatalf("calc next bit failed, index: %d, expected: %d, have: %d", i, c.bits, bits)
		}
	}
}

func TestCalcNextSeed(t *testing.T) {
	cases := []struct {
		node *BlockNode
		seed *bc.Hash
	}{
		{
			node: &BlockNode{
				Height: 0,
			},
			seed: consensus.InitialSeed,
		},
		{
			node: &BlockNode{
				Height: consensus.SeedPerRetarget - 1,
				Seed:   &bc.Hash{V1: 100},
			},
			seed: &bc.Hash{V1: 100},
		},
		{
			node: &BlockNode{
				Height: consensus.SeedPerRetarget,
				Seed:   &bc.Hash{V2: 200},
				Hash:   bc.Hash{V3: 300},
			},
			seed: &bc.Hash{V3: 300},
		},
	}

	for i, c := range cases {
		seed := c.node.CalcNextSeed()
		if *seed != *c.seed {
			t.Fatalf("calc next seed failed, index: %d, expected: %v, have: %v", i, c.seed, seed)
		}
	}
}

func TestSetMainChain(t *testing.T) {
	blockIndex := NewBlockIndex()
	var lastNode *BlockNode
	for i := uint64(0); i < 4; i++ {
		node := &BlockNode{
			Height: i,
			Hash:   bc.Hash{V0: i},
			Parent: lastNode,
		}
		blockIndex.AddNode(node)
		lastNode = node
	}

	tailNode := lastNode
	blockIndex.SetMainChain(lastNode)
	for lastNode.Parent != nil {
		if !blockIndex.InMainchain(lastNode.Hash) {
			t.Fatalf("block %d, hash %v is not in main chain", lastNode.Height, lastNode.Hash)
		}
		lastNode = lastNode.Parent
	}

	// fork and set main chain
	forkHeight := uint64(1)
	lastNode = blockIndex.nodeByHeight(forkHeight)
	for i := uint64(1); i <= 3; i++ {
		node := &BlockNode{
			Height: lastNode.Height + 1,
			Hash:   bc.Hash{V1: uint64(i)},
			Parent: lastNode,
		}
		blockIndex.AddNode(node)
		lastNode = node
	}

	bestNode := lastNode
	blockIndex.SetMainChain(lastNode)
	for lastNode.Parent != nil {
		if !blockIndex.InMainchain(lastNode.Hash) {
			t.Fatalf("after fork, block %d, hash %v is not in main chain", lastNode.Height, lastNode.Hash)
		}
		lastNode = lastNode.Parent
	}

	if bestNode != blockIndex.BestNode() {
		t.Fatalf("check best node failed")
	}

	for tailNode.Parent != nil && tailNode.Height > forkHeight {
		if blockIndex.InMainchain(tailNode.Hash) {
			t.Fatalf("old chain block %d, hash %v still in main chain", tailNode.Height, tailNode.Hash)
		}
		tailNode = tailNode.Parent
	}
}
