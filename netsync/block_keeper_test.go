package netsync

import (
	"container/list"
	"testing"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/test/mock"
	"github.com/bytom/testutil"
)

func TestAppendHeaderList(t *testing.T) {
	blocks := mockBlocks(7)
	cases := []struct {
		originalHeaders []*types.BlockHeader
		inputHeaders    []*types.BlockHeader
		wantHeaders     []*types.BlockHeader
		err             error
	}{
		{
			originalHeaders: []*types.BlockHeader{&blocks[0].BlockHeader},
			inputHeaders:    []*types.BlockHeader{&blocks[1].BlockHeader, &blocks[2].BlockHeader},
			wantHeaders:     []*types.BlockHeader{&blocks[0].BlockHeader, &blocks[1].BlockHeader, &blocks[2].BlockHeader},
			err:             nil,
		},
		{
			originalHeaders: []*types.BlockHeader{&blocks[5].BlockHeader},
			inputHeaders:    []*types.BlockHeader{&blocks[6].BlockHeader},
			wantHeaders:     []*types.BlockHeader{&blocks[5].BlockHeader, &blocks[6].BlockHeader},
			err:             nil,
		},
		{
			originalHeaders: []*types.BlockHeader{&blocks[5].BlockHeader},
			inputHeaders:    []*types.BlockHeader{&blocks[7].BlockHeader},
			wantHeaders:     []*types.BlockHeader{&blocks[5].BlockHeader},
			err:             errAppendHeaders,
		},
		{
			originalHeaders: []*types.BlockHeader{&blocks[5].BlockHeader},
			inputHeaders:    []*types.BlockHeader{&blocks[7].BlockHeader, &blocks[6].BlockHeader},
			wantHeaders:     []*types.BlockHeader{&blocks[5].BlockHeader},
			err:             errAppendHeaders,
		},
		{
			originalHeaders: []*types.BlockHeader{&blocks[2].BlockHeader},
			inputHeaders:    []*types.BlockHeader{&blocks[3].BlockHeader, &blocks[4].BlockHeader, &blocks[6].BlockHeader},
			wantHeaders:     []*types.BlockHeader{&blocks[2].BlockHeader, &blocks[3].BlockHeader, &blocks[4].BlockHeader},
			err:             errAppendHeaders,
		},
	}

	for i, c := range cases {
		bk := &blockKeeper{headerList: list.New()}
		for _, header := range c.originalHeaders {
			bk.headerList.PushBack(header)
		}

		if err := bk.appendHeaderList(c.inputHeaders); err != c.err {
			t.Errorf("case %d: got error %v want error %v", i, err, c.err)
		}

		gotHeaders := []*types.BlockHeader{}
		for e := bk.headerList.Front(); e != nil; e = e.Next() {
			gotHeaders = append(gotHeaders, e.Value.(*types.BlockHeader))
		}

		if !testutil.DeepEqual(gotHeaders, c.wantHeaders) {
			t.Errorf("case %d: got %v want %v", i, gotHeaders, c.wantHeaders)
		}
	}
}

func TestBlockLocator(t *testing.T) {
	blocks := mockBlocks(500)
	cases := []struct {
		bestHeight uint64
		wantHeight []uint64
	}{
		{
			bestHeight: 0,
			wantHeight: []uint64{0},
		},
		{
			bestHeight: 1,
			wantHeight: []uint64{1, 0},
		},
		{
			bestHeight: 7,
			wantHeight: []uint64{7, 6, 5, 4, 3, 2, 1, 0},
		},
		{
			bestHeight: 10,
			wantHeight: []uint64{10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
		},
		{
			bestHeight: 100,
			wantHeight: []uint64{100, 99, 98, 97, 96, 95, 94, 93, 92, 91, 89, 85, 77, 61, 29, 0},
		},
		{
			bestHeight: 500,
			wantHeight: []uint64{500, 499, 498, 497, 496, 495, 494, 493, 492, 491, 489, 485, 477, 461, 429, 365, 237, 0},
		},
	}

	for i, c := range cases {
		mockChain := mock.NewChain()
		bk := &blockKeeper{chain: mockChain}
		mockChain.SetBestBlockHeader(&blocks[c.bestHeight].BlockHeader)
		for i := uint64(0); i <= c.bestHeight; i++ {
			mockChain.SetBlockByHeight(i, blocks[i])
		}

		want := []*bc.Hash{}
		for _, i := range c.wantHeight {
			hash := blocks[i].Hash()
			want = append(want, &hash)
		}

		got := bk.blockLocator()
		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func mockBlocks(height uint64) []*types.Block {
	blocks := []*types.Block{}
	preHash := bc.Hash{}
	for i := uint64(0); i <= height; i++ {
		block := &types.Block{
			BlockHeader: types.BlockHeader{
				Height:            i,
				PreviousBlockHash: preHash,
			},
		}
		blocks = append(blocks, block)
		preHash = block.Hash()
	}
	return blocks
}
