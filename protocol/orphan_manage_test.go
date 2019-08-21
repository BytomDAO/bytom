package protocol

import (
	"testing"
	"time"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/testutil"
)

var testBlocks = []*types.Block{
	{BlockHeader: types.BlockHeader{
		PreviousBlockHash: bc.Hash{V0: 1},
		Nonce:             0,
	}},
	{BlockHeader: types.BlockHeader{
		PreviousBlockHash: bc.Hash{V0: 1},
		Nonce:             1,
	}},
	{BlockHeader: types.BlockHeader{
		PreviousBlockHash: bc.Hash{V0: 2},
		Nonce:             3,
	}},
}

var blockHashes = []bc.Hash{}

func init() {
	for _, block := range testBlocks {
		blockHashes = append(blockHashes, block.Hash())
	}
}

func TestDeleteLRU(t *testing.T) {
	now := time.Now()
	cases := []struct {
		before *OrphanManage
		after  *OrphanManage
	}{
		{
			before: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], now},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			after: &OrphanManage{
				orphan:      map[bc.Hash]*OrphanBlock{},
				prevOrphans: map[bc.Hash][]*bc.Hash{},
			},
		},
		{
			before: &OrphanManage{
				orphan:      map[bc.Hash]*OrphanBlock{},
				prevOrphans: map[bc.Hash][]*bc.Hash{},
			},
			after: &OrphanManage{
				orphan:      map[bc.Hash]*OrphanBlock{},
				prevOrphans: map[bc.Hash][]*bc.Hash{},
			},
		},
		{
			before: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], now.Add(2)},
					blockHashes[1]: {testBlocks[1], now.Add(1)},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0], &blockHashes[1]},
				},
			},
			after: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], now.Add(2)},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
		},
	}

	for i, c := range cases {
		c.before.deleteLRU()
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestOrphanManageAdd(t *testing.T) {
	cases := []struct {
		before    *OrphanManage
		after     *OrphanManage
		addOrphan *types.Block
	}{
		{
			before: &OrphanManage{
				orphan:      map[bc.Hash]*OrphanBlock{},
				prevOrphans: map[bc.Hash][]*bc.Hash{},
			},
			after: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			addOrphan: testBlocks[0],
		},
		{
			before: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			after: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			addOrphan: testBlocks[0],
		},
		{
			before: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			after: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
					blockHashes[1]: {testBlocks[1], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0], &blockHashes[1]},
				},
			},
			addOrphan: testBlocks[1],
		},
		{
			before: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			after: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
					blockHashes[2]: {testBlocks[2], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
					{V0: 2}: {&blockHashes[2]},
				},
			},
			addOrphan: testBlocks[2],
		},
	}

	for i, c := range cases {
		c.before.Add(c.addOrphan)
		for _, orphan := range c.before.orphan {
			orphan.expiration = time.Time{}
		}
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestOrphanManageDelete(t *testing.T) {
	cases := []struct {
		before *OrphanManage
		after  *OrphanManage
		remove *bc.Hash
	}{
		{
			before: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			after: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			remove: &blockHashes[1],
		},
		{
			before: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			after: &OrphanManage{
				orphan:      map[bc.Hash]*OrphanBlock{},
				prevOrphans: map[bc.Hash][]*bc.Hash{},
			},
			remove: &blockHashes[0],
		},
		{
			before: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
					blockHashes[1]: {testBlocks[1], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0], &blockHashes[1]},
				},
			},
			after: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {testBlocks[0], time.Time{}},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			remove: &blockHashes[1],
		},
	}

	for i, c := range cases {
		c.before.delete(c.remove)
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestOrphanManageExpire(t *testing.T) {
	cases := []struct {
		before *OrphanManage
		after  *OrphanManage
	}{
		{
			before: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {
						testBlocks[0],
						time.Unix(1633479700, 0),
					},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			after: &OrphanManage{
				orphan:      map[bc.Hash]*OrphanBlock{},
				prevOrphans: map[bc.Hash][]*bc.Hash{},
			},
		},
		{
			before: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {
						testBlocks[0],
						time.Unix(1633479702, 0),
					},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
			after: &OrphanManage{
				orphan: map[bc.Hash]*OrphanBlock{
					blockHashes[0]: {
						testBlocks[0],
						time.Unix(1633479702, 0),
					},
				},
				prevOrphans: map[bc.Hash][]*bc.Hash{
					{V0: 1}: {&blockHashes[0]},
				},
			},
		},
	}

	for i, c := range cases {
		c.before.orphanExpire(time.Unix(1633479701, 0))
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestOrphanManageNumLimit(t *testing.T) {
	cases := []struct {
		addOrphanBlockNum    int
		expectOrphanBlockNum int
	}{
		{
			addOrphanBlockNum:    10,
			expectOrphanBlockNum: 10,
		},
		{
			addOrphanBlockNum:    numOrphanBlockLimit,
			expectOrphanBlockNum: numOrphanBlockLimit,
		},
		{
			addOrphanBlockNum:    numOrphanBlockLimit + 1,
			expectOrphanBlockNum: numOrphanBlockLimit,
		},
		{
			addOrphanBlockNum:    numOrphanBlockLimit + 10,
			expectOrphanBlockNum: numOrphanBlockLimit,
		},
	}

	for i, c := range cases {
		orphanManage := &OrphanManage{
			orphan:      map[bc.Hash]*OrphanBlock{},
			prevOrphans: map[bc.Hash][]*bc.Hash{},
		}
		for num := 0; num < c.addOrphanBlockNum; num++ {
			orphanManage.Add(&types.Block{BlockHeader: types.BlockHeader{Height: uint64(num)}})
		}
		if len(orphanManage.orphan) != c.expectOrphanBlockNum {
			t.Errorf("case %d: got %d want %d", i, len(orphanManage.orphan), c.expectOrphanBlockNum)
		}
	}
}
