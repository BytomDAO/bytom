package protocol

import (
	"testing"

	"github.com/bytom/config"
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
