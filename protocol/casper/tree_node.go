package casper

import (
	"errors"
	"fmt"

	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/state"
)

type treeNode struct {
	*state.Checkpoint
	children []*treeNode
}

func makeTree(root *state.Checkpoint, successors []*state.Checkpoint) *treeNode {
	parentToSuccessors := make(map[bc.Hash][]*state.Checkpoint)
	for _, successor := range successors {
		parentToSuccessors[successor.ParentHash] = append(parentToSuccessors[successor.ParentHash], successor)
	}

	rootNode := &treeNode{Checkpoint: root}
	nodes := []*treeNode{rootNode}
	for len(nodes) != 0 {
		node := nodes[0]
		for _, successor := range parentToSuccessors[node.Hash] {
			child := &treeNode{Checkpoint: successor}
			successor.Parent = node.Checkpoint
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

func (t *treeNode) bestNode(justifiedHeight uint64) (*treeNode, uint64) {
	if t.Status == state.Justified {
		justifiedHeight = t.Height
	}

	bestNode, bestJustified := t, justifiedHeight
	for _, child := range t.children {
		bestChild, childJustified := child.bestNode(justifiedHeight)
		if childJustified > bestJustified || (childJustified == bestJustified && bestChild.Height > bestNode.Height) ||
			(childJustified == bestJustified && bestChild.Height == bestNode.Height && bestChild.Hash.String() > bestNode.Hash.String()) {
			bestNode, bestJustified = bestChild, childJustified
		}
	}

	return bestNode, bestJustified
}

func (t *treeNode) lastJustified() *treeNode {
	var selected *treeNode
	if t.Status == state.Justified {
		selected = t
	}

	for _, child := range t.children {
		if childSelected := child.lastJustified(); childSelected == nil {
			continue
		} else if selected == nil || childSelected.Height > selected.Height {
			selected = childSelected
		}
	}
	return selected
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
	if predicate(t.Checkpoint) {
		return t
	}

	for _, child := range t.children {
		if node := child.findOnlyOne(predicate); node != nil {
			return node
		}
	}
	return nil
}
