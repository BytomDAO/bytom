package consensus

import (
	"errors"
	"fmt"

	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/state"
)

type treeNode struct {
	checkpoint *state.Checkpoint
	children   []*treeNode
}

func (t *treeNode) nodeByHash(blockHash bc.Hash) (*treeNode, error) {
	if c := t.findOnlyOne(func(c *state.Checkpoint) bool {
		return c.Hash == blockHash
	}); c != nil {
		return c, nil
	}

	return nil, errors.New(fmt.Sprintf("fail to find checkpoint of hash:%s", blockHash.String()))
}

func (t *treeNode) findOnlyOne(predicate func(*state.Checkpoint) bool) *treeNode {
	if predicate(t.checkpoint) {
		return t
	}

	for _, child := range t.children {
		if node := child.findOnlyOne(predicate); node != nil {
			return node
		}
	}
	return nil
}
