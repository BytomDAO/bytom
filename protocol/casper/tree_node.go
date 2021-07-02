package casper

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

func makeTree(root *state.Checkpoint, successors []*state.Checkpoint) *treeNode {
	parentToSuccessors := make(map[bc.Hash][]*state.Checkpoint)
	for _, successor := range successors {
		parentToSuccessors[successor.ParentHash] = append(parentToSuccessors[successor.ParentHash], successor)
	}

	rootNode := &treeNode{checkpoint: root}
	nodes := []*treeNode{rootNode}
	for len(nodes) != 0 {
		node := nodes[0]
		for _, successor := range parentToSuccessors[node.checkpoint.Hash] {
			child := &treeNode{checkpoint: successor}
			successor.Parent = node.checkpoint
			node.addChild(child)
			nodes = append(nodes, child)
		}
		nodes = nodes[1:]
	}
	return rootNode
}

func (t *treeNode) addChild(child *treeNode) {
	t.children = append(t.children, child)
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
