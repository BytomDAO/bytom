package protocol

import (
	"testing"

	"github.com/bytom/config"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/state"
	"github.com/bytom/testutil"
)

func TestCalcReorganizeNodes(t *testing.T) {
	c := &Chain{index: state.NewBlockIndex()}
	header := config.GenesisBlock().BlockHeader
	initNode, err := state.NewBlockNode(&header, nil)
	if err != nil {
		t.Fatal(err)
	}

	c.index.AddNode(initNode)
	var wantAttachNodes []*state.BlockNode
	var wantDetachNodes []*state.BlockNode

	mainChainNode := initNode
	for i := 1; i <= 7; i++ {
		header.Height = uint64(i)
		header.Nonce = 0
		mainChainNode, err = state.NewBlockNode(&header, mainChainNode)
		if err != nil {
			t.Fatal(err)
		}
		wantDetachNodes = append([]*state.BlockNode{mainChainNode}, wantDetachNodes...)
		c.index.AddNode(mainChainNode)
	}
	c.bestNode = mainChainNode
	c.index.SetMainChain(mainChainNode)

	sideChainNode := initNode
	for i := 1; i <= 13; i++ {
		header.Height = uint64(i)
		header.Nonce = 1
		sideChainNode, err = state.NewBlockNode(&header, sideChainNode)
		if err != nil {
			t.Fatal(err)
		}
		wantAttachNodes = append(wantAttachNodes, sideChainNode)
		c.index.AddNode(sideChainNode)
	}

	getAttachNodes, getDetachNodes := c.calcReorganizeNodes(sideChainNode)
	if !testutil.DeepEqual(wantAttachNodes, getAttachNodes) {
		t.Errorf("attach nodes want %v but get %v", wantAttachNodes, getAttachNodes)
	}
	if !testutil.DeepEqual(wantDetachNodes, getDetachNodes) {
		t.Errorf("detach nodes want %v but get %v", wantDetachNodes, getDetachNodes)
	}
}

func TestEdgeCalcReorganizeNodes(t *testing.T) {
	header := config.GenesisBlock().BlockHeader
	initNode, err := state.NewBlockNode(&header, nil)
	if err != nil {
		t.Fatal(err)
	}

	testNodes := []*state.BlockNode{initNode}
	testNewNodes := []*state.BlockNode{initNode}
	for i := uint64(1); i <= 5; i++ {
		node := &state.BlockNode{
			Height: i,
			Nonce:  0,
			Hash:   bc.Hash{V0: uint64(i)},
			Parent: testNodes[i-1],
		}
		testNodes = append(testNodes, node)

		newNode := &state.BlockNode{
			Height: i,
			Nonce:  1,
			Hash:   bc.Hash{V1: uint64(i)},
			Parent: testNewNodes[i-1],
		}
		testNewNodes = append(testNewNodes, newNode)
	}

	cases := []struct {
		mainChainNode   *state.BlockNode
		newNode         *state.BlockNode
		wantAttachNodes []*state.BlockNode
		wantDetachNodes []*state.BlockNode
	}{
		{
			mainChainNode:   testNodes[1],
			newNode:         testNodes[5],
			wantAttachNodes: testNodes[2:],
			wantDetachNodes: []*state.BlockNode{},
		},
		{
			mainChainNode:   testNodes[5],
			newNode:         testNodes[2],
			wantAttachNodes: []*state.BlockNode{},
			wantDetachNodes: []*state.BlockNode{testNodes[5], testNodes[4], testNodes[3]},
		},
		{
			mainChainNode:   testNodes[2],
			newNode:         testNodes[2],
			wantAttachNodes: []*state.BlockNode{},
			wantDetachNodes: []*state.BlockNode{},
		},
		{
			mainChainNode:   testNewNodes[3],
			newNode:         testNodes[2],
			wantAttachNodes: testNodes[1:3],
			wantDetachNodes: []*state.BlockNode{testNewNodes[3], testNewNodes[2], testNewNodes[1]},
		},
		{
			mainChainNode:   testNewNodes[2],
			newNode:         testNodes[3],
			wantAttachNodes: testNodes[1:4],
			wantDetachNodes: []*state.BlockNode{testNewNodes[2], testNewNodes[1]},
		},
		{
			mainChainNode:   testNodes[5],
			newNode:         testNewNodes[3],
			wantAttachNodes: testNewNodes[1:4],
			wantDetachNodes: []*state.BlockNode{testNodes[5], testNodes[4], testNodes[3], testNodes[2], testNodes[1]},
		},
	}

	for i, c := range cases {
		chain := &Chain{index: state.NewBlockIndex()}
		chain.index.AddNode(initNode)
		for i := uint64(1); i <= c.mainChainNode.Height; i++ {
			chain.index.AddNode(testNodes[i])
		}
		chain.bestNode = c.mainChainNode
		chain.index.SetMainChain(c.mainChainNode)
		getAttachNodes, getDetachNodes := chain.calcReorganizeNodes(c.newNode)

		if !testutil.DeepEqual(c.wantAttachNodes, getAttachNodes) {
			t.Errorf("test case %d, attach nodes want %v but get %v", i, c.wantAttachNodes, getAttachNodes)
		}

		if !testutil.DeepEqual(c.wantDetachNodes, getDetachNodes) {
			t.Errorf("test case %d, detach nodes want %v but get %v", i, c.wantDetachNodes, getDetachNodes)
		}
	}
}
