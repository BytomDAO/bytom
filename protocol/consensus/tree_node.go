package consensus

import (
	"errors"
	"fmt"
)

type treeNode struct {
	checkpoint *checkpoint
	children   []*treeNode
}

func (t *treeNode) checkpointByHash(blockHash string) (*checkpoint, error) {
	if c := t.findOnlyOne(func(c *checkpoint) bool {
		return c.hash == blockHash
	}); c != nil {
		return c, nil
	}

	return nil, errors.New(fmt.Sprintf("fail to find checkpoint of hash:%s", blockHash))
}

func (t *treeNode) checkpointsOfHeight(blockHeight uint64) []*checkpoint {
	if blockHeight%blocksOfEpoch != 0 {
		return nil
	}

	var result []*checkpoint
	if t.checkpoint.height == blockHeight {
		return append(result, t.checkpoint)
	}

	for _, child := range t.children {
		result = append(result, child.checkpointsOfHeight(blockHeight)...)
	}
	return result
}

func (t *treeNode) findOnlyOne(predicate func(*checkpoint) bool) *checkpoint {
	if predicate(t.checkpoint) {
		return t.checkpoint
	}

	for _, child := range t.children {
		if c := child.findOnlyOne(predicate); c != nil {
			return c
		}
	}
	return nil
}
