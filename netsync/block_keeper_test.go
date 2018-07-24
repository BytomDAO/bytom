package netsync

import (
	"container/list"
	"testing"
	"time"

	"github.com/bytom/consensus"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/test/mock"
	"github.com/bytom/testutil"
)

func TestAppendHeaderList(t *testing.T) {
	blocks := mockBlocks(nil, 7)
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
	blocks := mockBlocks(nil, 500)
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

		if got := bk.blockLocator(); !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func TestFastBlockSync(t *testing.T) {
	maxBlockPerMsg = 5
	maxBlockHeadersPerMsg = 10
	baseChain := mockBlocks(nil, 300)

	cases := []struct {
		syncTimeout time.Duration
		aBlocks     []*types.Block
		bBlocks     []*types.Block
		checkPoint  *consensus.Checkpoint
		want        []*types.Block
		err         error
	}{
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:100],
			bBlocks:     baseChain[:301],
			checkPoint: &consensus.Checkpoint{
				Height: baseChain[250].Height,
				Hash:   baseChain[250].Hash(),
			},
			want: baseChain[:251],
			err:  nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:100],
			bBlocks:     baseChain[:301],
			checkPoint: &consensus.Checkpoint{
				Height: baseChain[100].Height,
				Hash:   baseChain[100].Hash(),
			},
			want: baseChain[:101],
			err:  nil,
		},
		{
			syncTimeout: 1 * time.Millisecond,
			aBlocks:     baseChain[:100],
			bBlocks:     baseChain[:100],
			checkPoint: &consensus.Checkpoint{
				Height: baseChain[200].Height,
				Hash:   baseChain[200].Hash(),
			},
			want: baseChain[:100],
			err:  errRequestTimeout,
		},
	}

	for i, c := range cases {
		syncTimeout = c.syncTimeout
		a := mockSync(c.aBlocks)
		b := mockSync(c.bBlocks)
		netWork := NewNetWork()
		netWork.Register(a, "192.168.0.1", "test node A", consensus.SFFullNode)
		netWork.Register(b, "192.168.0.2", "test node B", consensus.SFFullNode)
		if err := netWork.HandsShake(a, b); err != nil {
			t.Errorf("fail on peer hands shake %v", err)
		}

		a.blockKeeper.syncPeer = a.peers.getPeer("test node B")
		if err := a.blockKeeper.fastBlockSync(c.checkPoint); errors.Root(err) != c.err {
			t.Errorf("case %d: got %v want %v", i, err, c.err)
		}

		got := []*types.Block{}
		for i := uint64(0); i <= a.chain.BestBlockHeight(); i++ {
			block, err := a.chain.GetBlockByHeight(i)
			if err != nil {
				t.Error("case %d got err %v", err)
			}
			got = append(got, block)
		}

		if !testutil.DeepEqual(got, c.want) {
			t.Errorf("case %d: got %v want %v", i, got, c.want)
		}
	}
}

func TestLocateBlocks(t *testing.T) {
	maxBlockPerMsg = 5
	blocks := mockBlocks(nil, 100)
	cases := []struct {
		locator    []uint64
		stopHash   bc.Hash
		wantHeight []uint64
	}{
		{
			locator:    []uint64{20},
			stopHash:   blocks[100].Hash(),
			wantHeight: []uint64{21, 22, 23, 24, 25},
		},
	}

	mockChain := mock.NewChain()
	bk := &blockKeeper{chain: mockChain}
	for _, block := range blocks {
		mockChain.SetBlockByHeight(block.Height, block)
	}

	for i, c := range cases {
		locator := []*bc.Hash{}
		for _, i := range c.locator {
			hash := blocks[i].Hash()
			locator = append(locator, &hash)
		}

		want := []*types.Block{}
		for _, i := range c.wantHeight {
			want = append(want, blocks[i])
		}

		got, _ := bk.locateBlocks(locator, &c.stopHash)
		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func TestLocateHeaders(t *testing.T) {
	maxBlockHeadersPerMsg = 10
	blocks := mockBlocks(nil, 150)
	cases := []struct {
		chainHeight uint64
		locator     []uint64
		stopHash    bc.Hash
		wantHeight  []uint64
		err         bool
	}{
		{
			chainHeight: 100,
			locator:     []uint64{},
			stopHash:    blocks[100].Hash(),
			wantHeight:  []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    blocks[100].Hash(),
			wantHeight:  []uint64{21, 22, 23, 24, 25, 26, 27, 28, 29, 30},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    blocks[24].Hash(),
			wantHeight:  []uint64{21, 22, 23, 24},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    blocks[20].Hash(),
			wantHeight:  []uint64{},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    bc.Hash{},
			wantHeight:  []uint64{},
			err:         true,
		},
		{
			chainHeight: 100,
			locator:     []uint64{120, 70},
			stopHash:    blocks[78].Hash(),
			wantHeight:  []uint64{71, 72, 73, 74, 75, 76, 77, 78},
			err:         false,
		},
	}

	for i, c := range cases {
		mockChain := mock.NewChain()
		bk := &blockKeeper{chain: mockChain}
		for i := uint64(0); i <= c.chainHeight; i++ {
			mockChain.SetBlockByHeight(i, blocks[i])
		}

		locator := []*bc.Hash{}
		for _, i := range c.locator {
			hash := blocks[i].Hash()
			locator = append(locator, &hash)
		}

		want := []*types.BlockHeader{}
		for _, i := range c.wantHeight {
			want = append(want, &blocks[i].BlockHeader)
		}

		got, err := bk.locateHeaders(locator, &c.stopHash)
		if err != nil != c.err {
			t.Errorf("case %d: got %v want err = %v", i, err, c.err)
		}
		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func TestNextCheckpoint(t *testing.T) {
	cases := []struct {
		checkPoints []consensus.Checkpoint
		bestHeight  uint64
		want        *consensus.Checkpoint
	}{
		{
			checkPoints: []consensus.Checkpoint{},
			bestHeight:  5000,
			want:        nil,
		},
		{
			checkPoints: []consensus.Checkpoint{
				{10000, bc.Hash{V0: 1}},
			},
			bestHeight: 5000,
			want:       &consensus.Checkpoint{10000, bc.Hash{V0: 1}},
		},
		{
			checkPoints: []consensus.Checkpoint{
				{10000, bc.Hash{V0: 1}},
				{20000, bc.Hash{V0: 2}},
				{30000, bc.Hash{V0: 3}},
			},
			bestHeight: 15000,
			want:       &consensus.Checkpoint{20000, bc.Hash{V0: 2}},
		},
		{
			checkPoints: []consensus.Checkpoint{
				{10000, bc.Hash{V0: 1}},
				{20000, bc.Hash{V0: 2}},
				{30000, bc.Hash{V0: 3}},
			},
			bestHeight: 10000,
			want:       &consensus.Checkpoint{20000, bc.Hash{V0: 2}},
		},
		{
			checkPoints: []consensus.Checkpoint{
				{10000, bc.Hash{V0: 1}},
				{20000, bc.Hash{V0: 2}},
				{30000, bc.Hash{V0: 3}},
			},
			bestHeight: 35000,
			want:       nil,
		},
	}

	mockChain := mock.NewChain()
	for i, c := range cases {
		consensus.ActiveNetParams.Checkpoints = c.checkPoints
		mockChain.SetBestBlockHeader(&types.BlockHeader{Height: c.bestHeight})
		bk := &blockKeeper{chain: mockChain}

		if got := bk.nextCheckpoint(); !testutil.DeepEqual(got, c.want) {
			t.Errorf("case %d: got %v want %v", i, got, c.want)
		}
	}
}

func TestRegularBlockSync(t *testing.T) {
	baseChain := mockBlocks(nil, 50)
	chainX := append(baseChain, mockBlocks(baseChain[50], 60)...)
	chainY := append(baseChain, mockBlocks(baseChain[50], 70)...)
	cases := []struct {
		syncTimeout time.Duration
		aBlocks     []*types.Block
		bBlocks     []*types.Block
		syncHeight  uint64
		want        []*types.Block
		err         error
	}{
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:20],
			bBlocks:     baseChain[:50],
			syncHeight:  45,
			want:        baseChain[:46],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     chainX,
			bBlocks:     chainY,
			syncHeight:  70,
			want:        chainY,
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     chainX[:52],
			bBlocks:     chainY[:53],
			syncHeight:  52,
			want:        chainY[:53],
			err:         nil,
		},
		{
			syncTimeout: 1 * time.Millisecond,
			aBlocks:     baseChain,
			bBlocks:     baseChain,
			syncHeight:  52,
			want:        baseChain,
			err:         errRequestTimeout,
		},
	}

	for i, c := range cases {
		syncTimeout = c.syncTimeout
		a := mockSync(c.aBlocks)
		b := mockSync(c.bBlocks)
		netWork := NewNetWork()
		netWork.Register(a, "192.168.0.1", "test node A", consensus.SFFullNode)
		netWork.Register(b, "192.168.0.2", "test node B", consensus.SFFullNode)
		if err := netWork.HandsShake(a, b); err != nil {
			t.Errorf("fail on peer hands shake %v", err)
		}

		a.blockKeeper.syncPeer = a.peers.getPeer("test node B")
		if err := a.blockKeeper.regularBlockSync(c.syncHeight); errors.Root(err) != c.err {
			t.Errorf("case %d: got %v want %v", i, err, c.err)
		}

		got := []*types.Block{}
		for i := uint64(0); i <= a.chain.BestBlockHeight(); i++ {
			block, err := a.chain.GetBlockByHeight(i)
			if err != nil {
				t.Error("case %d got err %v", err)
			}
			got = append(got, block)
		}

		if !testutil.DeepEqual(got, c.want) {
			t.Errorf("case %d: got %v want %v", i, got, c.want)
		}
	}
}

func TestRequireBlock(t *testing.T) {
	blocks := mockBlocks(nil, 5)
	a := mockSync(blocks[:1])
	b := mockSync(blocks[:5])
	netWork := NewNetWork()
	netWork.Register(a, "192.168.0.1", "test node A", consensus.SFFullNode)
	netWork.Register(b, "192.168.0.2", "test node B", consensus.SFFullNode)
	if err := netWork.HandsShake(a, b); err != nil {
		t.Errorf("fail on peer hands shake %v", err)
	}

	a.blockKeeper.syncPeer = a.peers.getPeer("test node B")
	b.blockKeeper.syncPeer = b.peers.getPeer("test node A")
	cases := []struct {
		syncTimeout   time.Duration
		testNode      *SyncManager
		requireHeight uint64
		want          *types.Block
		err           error
	}{
		{
			syncTimeout:   30 * time.Second,
			testNode:      a,
			requireHeight: 4,
			want:          blocks[4],
			err:           nil,
		},
		{
			syncTimeout:   1 * time.Millisecond,
			testNode:      b,
			requireHeight: 4,
			want:          nil,
			err:           errRequestTimeout,
		},
	}

	for i, c := range cases {
		syncTimeout = c.syncTimeout
		got, err := c.testNode.blockKeeper.requireBlock(c.requireHeight)
		if !testutil.DeepEqual(got, c.want) {
			t.Errorf("case %d: got %v want %v", i, got, c.want)
		}
		if errors.Root(err) != c.err {
			t.Errorf("case %d: got %v want %v", i, err, c.err)
		}
	}
}
