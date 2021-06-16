package protocol

import (
	"testing"

	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
	"github.com/bytom/bytom/testutil"
)

func TestMergeCheckpoint(t *testing.T) {
	parent := &treeNode{
		checkpoint: &state.Checkpoint{
			Height:          0,
			Hash:            testutil.MustDecodeHash("a4de8b26e4394ebc5f63e9c805cc682ae6fa47df141d4fe2a238d7c11ccbd87f"),
			ParentHash:      bc.Hash{},
			Parent:          nil,
			Timestamp:       1563606689,
			Status:          state.Justified,
		},
	}

	block := &types.Block{
		BlockHeader:  types.BlockHeader{
			Height:   1,
			PreviousBlockHash: testutil.MustDecodeHash("a4de8b26e4394ebc5f63e9c805cc682ae6fa47df141d4fe2a238d7c11ccbd87f"),
			Timestamp:         1563606700,
		},
	}

	child1 := &treeNode{checkpoint: state.NewCheckpoint(parent.checkpoint)}
	parent.addChild(child1)
	if err := child1.checkpoint.Increase(block); err != nil {
		t.Fatal(err)
	}

	child2 := &treeNode{checkpoint: state.NewCheckpoint(parent.checkpoint)}
	parent.addChild(child2)
	if err := child2.checkpoint.Increase(block); err != nil {
		t.Fatal(err)
	}

	if len(parent.children) != 1 {
		t.Errorf("expect size of children is 1, got %d\n", len(parent.children))
	}
}
