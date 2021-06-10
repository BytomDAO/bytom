package chainmgr

import (
	"io/ioutil"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/bytom/bytom/consensus"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/test/mock"
	"github.com/bytom/bytom/testcontrol"
	"github.com/bytom/bytom/testutil"
)

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
		fs := &fastSync{chain: mockChain}
		mockChain.SetBestBlockHeader(&blocks[c.bestHeight].BlockHeader)
		for i := uint64(0); i <= c.bestHeight; i++ {
			mockChain.SetBlockByHeight(i, blocks[i])
		}

		want := []*bc.Hash{}
		for _, i := range c.wantHeight {
			hash := blocks[i].Hash()
			want = append(want, &hash)
		}

		if got := fs.blockLocator(); !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func TestFastBlockSync(t *testing.T) {
	if testcontrol.IgnoreTestTemporary {
		return
	}

	tmp, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatalf("failed to create temporary data folder: %v", err)
	}
	testDBA := dbm.NewDB("testdba", "leveldb", tmp)
	testDBB := dbm.NewDB("testdbb", "leveldb", tmp)
	defer func() {
		testDBA.Close()
		testDBB.Close()
		os.RemoveAll(tmp)
	}()

	maxSizeOfSyncSkeleton = 11
	numOfBlocksSkeletonGap = 10
	maxNumOfBlocksPerSync = numOfBlocksSkeletonGap * uint64(maxSizeOfSyncSkeleton-1)
	fastSyncPivotGap = uint64(5)
	minGapStartFastSync = uint64(6)

	defer func() {
		maxSizeOfSyncSkeleton = 11
		numOfBlocksSkeletonGap = maxNumOfBlocksPerMsg
		maxNumOfBlocksPerSync = numOfBlocksSkeletonGap * uint64(maxSizeOfSyncSkeleton-1)
		fastSyncPivotGap = uint64(64)
		minGapStartFastSync = uint64(128)
		requireHeadersTimeout = 30 * time.Second
	}()

	baseChain := mockBlocks(nil, 300)
	chainX := []*types.Block{}
	chainX = append(chainX, baseChain[:30]...)
	chainX = append(chainX, mockBlocks(baseChain[30], 500)...)
	cases := []struct {
		syncTimeout time.Duration
		aBlocks     []*types.Block
		bBlocks     []*types.Block
		want        []*types.Block
		err         error
	}{
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:50],
			bBlocks:     baseChain[:301],
			want:        baseChain[:150],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:2],
			bBlocks:     baseChain[:300],
			want:        baseChain[:102],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:2],
			bBlocks:     baseChain[:53],
			want:        baseChain[:48],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:2],
			bBlocks:     baseChain[:53],
			want:        baseChain[:48],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:2],
			bBlocks:     baseChain[:10],
			want:        baseChain[:5],
			err:         nil,
		},
		{
			syncTimeout: 0 * time.Second,
			aBlocks:     baseChain[:50],
			bBlocks:     baseChain[:301],
			want:        baseChain[:50],
			err:         errSkeletonSize,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     chainX[:50],
			bBlocks:     baseChain[:301],
			want:        baseChain[:128],
			err:         nil,
		},
	}

	for i, c := range cases {
		a := mockSync(c.aBlocks, nil, testDBA)
		b := mockSync(c.bBlocks, nil, testDBB)
		netWork := NewNetWork()
		netWork.Register(a, "192.168.0.1", "test node A", consensus.SFFullNode|consensus.SFFastSync)
		netWork.Register(b, "192.168.0.2", "test node B", consensus.SFFullNode|consensus.SFFastSync)
		if B2A, A2B, err := netWork.HandsShake(a, b); err != nil {
			t.Errorf("fail on peer hands shake %v", err)
		} else {
			go B2A.postMan()
			go A2B.postMan()
		}
		a.blockKeeper.syncPeer = a.peers.GetPeer("test node B")
		a.blockKeeper.fastSync.setSyncPeer(a.blockKeeper.syncPeer)

		requireHeadersTimeout = c.syncTimeout
		if err := a.blockKeeper.fastSync.process(); errors.Root(err) != c.err {
			t.Errorf("case %d: got %v want %v", i, err, c.err)
		}

		got := []*types.Block{}
		for i := uint64(0); i <= a.chain.BestBlockHeight(); i++ {
			block, err := a.chain.GetBlockByHeight(i)
			if err != nil {
				t.Errorf("case %d got err %v", i, err)
			}
			got = append(got, block)
		}
		if !testutil.DeepEqual(got, c.want) {
			t.Errorf("case %d: got %v want %v", i, got, c.want)
		}
	}
}

type mockFetcher struct {
	baseChain  []*types.Block
	peerStatus map[string][]*types.Block
	peers      []string
	testType   int
}

func (mf *mockFetcher) resetParameter() {
	return
}

func (mf *mockFetcher) addSyncPeer(peerID string) {
	return
}

func (mf *mockFetcher) requireBlock(peerID string, height uint64) (*types.Block, error) {
	return nil, nil
}

func (mf *mockFetcher) parallelFetchBlocks(work []*fetchBlocksWork, downloadNotifyCh chan struct{}, ProcessStopCh chan struct{}, wg *sync.WaitGroup) {
	return
}

func (mf *mockFetcher) parallelFetchHeaders(peers []*peers.Peer, locator []*bc.Hash, stopHash *bc.Hash, skip uint64) map[string][]*types.BlockHeader {
	result := make(map[string][]*types.BlockHeader)
	switch mf.testType {
	case 1:
		result["peer1"] = []*types.BlockHeader{&mf.peerStatus["peer1"][1000].BlockHeader, &mf.peerStatus["peer1"][1100].BlockHeader, &mf.peerStatus["peer1"][1200].BlockHeader,
			&mf.peerStatus["peer1"][1300].BlockHeader, &mf.peerStatus["peer1"][1400].BlockHeader, &mf.peerStatus["peer1"][1500].BlockHeader,
			&mf.peerStatus["peer1"][1600].BlockHeader, &mf.peerStatus["peer1"][1700].BlockHeader, &mf.peerStatus["peer1"][1800].BlockHeader,
		}
		result["peer2"] = []*types.BlockHeader{&mf.peerStatus["peer2"][1000].BlockHeader, &mf.peerStatus["peer2"][1100].BlockHeader, &mf.peerStatus["peer2"][1200].BlockHeader,
			&mf.peerStatus["peer2"][1300].BlockHeader, &mf.peerStatus["peer2"][1400].BlockHeader, &mf.peerStatus["peer2"][1500].BlockHeader,
			&mf.peerStatus["peer2"][1600].BlockHeader, &mf.peerStatus["peer2"][1700].BlockHeader, &mf.peerStatus["peer2"][1800].BlockHeader,
		}

	case 2:
		result["peer1"] = []*types.BlockHeader{}
	case 3:
	case 4:
		result["peer2"] = []*types.BlockHeader{&mf.peerStatus["peer2"][1000].BlockHeader, &mf.peerStatus["peer2"][1100].BlockHeader, &mf.peerStatus["peer2"][1200].BlockHeader,
			&mf.peerStatus["peer2"][1300].BlockHeader, &mf.peerStatus["peer2"][1400].BlockHeader, &mf.peerStatus["peer2"][1500].BlockHeader,
			&mf.peerStatus["peer2"][1600].BlockHeader, &mf.peerStatus["peer2"][1700].BlockHeader, &mf.peerStatus["peer2"][1800].BlockHeader,
		}
	case 5:
		result["peer1"] = []*types.BlockHeader{&mf.peerStatus["peer1"][1000].BlockHeader, &mf.peerStatus["peer1"][1100].BlockHeader, &mf.peerStatus["peer1"][1200].BlockHeader,
			&mf.peerStatus["peer1"][1300].BlockHeader, &mf.peerStatus["peer1"][1400].BlockHeader, &mf.peerStatus["peer1"][1500].BlockHeader,
			&mf.peerStatus["peer1"][1600].BlockHeader, &mf.peerStatus["peer1"][1700].BlockHeader, &mf.peerStatus["peer1"][1800].BlockHeader,
		}
		result["peer2"] = []*types.BlockHeader{&mf.peerStatus["peer2"][1000].BlockHeader, &mf.peerStatus["peer2"][1100].BlockHeader, &mf.peerStatus["peer2"][1200].BlockHeader,
			&mf.peerStatus["peer2"][1300].BlockHeader, &mf.peerStatus["peer2"][1400].BlockHeader, &mf.peerStatus["peer2"][1500].BlockHeader,
			&mf.peerStatus["peer2"][1600].BlockHeader, &mf.peerStatus["peer2"][1700].BlockHeader,
		}
	}
	return result
}

func TestCreateFetchBlocksTasks(t *testing.T) {
	baseChain := mockBlocks(nil, 1000)
	chainX := append(baseChain, mockBlocks(baseChain[1000], 2000)...)
	chainY := append(baseChain, mockBlocks(baseChain[1000], 1900)...)
	peerStatus := make(map[string][]*types.Block)
	peerStatus["peer1"] = chainX
	peerStatus["peer2"] = chainY
	type syncPeer struct {
		peer               *P2PPeer
		bestHeight         uint64
		irreversibleHeight uint64
	}

	cases := []struct {
		peers        []*syncPeer
		mainSyncPeer string
		testType     int
		wantTasks    []*fetchBlocksWork
		wantErr      error
	}{
		// normal test
		{
			peers: []*syncPeer{
				{peer: &P2PPeer{id: "peer1", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 1000, irreversibleHeight: 1000},
				{peer: &P2PPeer{id: "peer2", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 800, irreversibleHeight: 800},
			},
			mainSyncPeer: "peer1",
			testType:     1,
			wantTasks: []*fetchBlocksWork{
				{&chainX[1000].BlockHeader, &chainX[1100].BlockHeader}, {&chainX[1100].BlockHeader, &chainX[1200].BlockHeader},
				{&chainX[1200].BlockHeader, &chainX[1300].BlockHeader}, {&chainX[1300].BlockHeader, &chainX[1400].BlockHeader},
				{&chainX[1400].BlockHeader, &chainX[1500].BlockHeader}, {&chainX[1500].BlockHeader, &chainX[1600].BlockHeader},
				{&chainX[1600].BlockHeader, &chainX[1700].BlockHeader}, {&chainX[1700].BlockHeader, &chainX[1800].BlockHeader},
			},
			wantErr: nil,
		},
		// test no sync peer
		{
			peers:        []*syncPeer{},
			mainSyncPeer: "peer1",
			testType:     0,
			wantTasks:    nil,
			wantErr:      errNoSyncPeer,
		},
		// primary sync peer skeleton size error
		{
			peers: []*syncPeer{
				{peer: &P2PPeer{id: "peer1", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 1000, irreversibleHeight: 1000},
				{peer: &P2PPeer{id: "peer2", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 800, irreversibleHeight: 800},
			},
			mainSyncPeer: "peer1",
			testType:     2,
			wantTasks:    nil,
			wantErr:      errSkeletonSize,
		},
		// no skeleton return
		{
			peers: []*syncPeer{
				{peer: &P2PPeer{id: "peer1", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 1000, irreversibleHeight: 1000},
				{peer: &P2PPeer{id: "peer2", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 800, irreversibleHeight: 800},
			},
			mainSyncPeer: "peer1",
			testType:     3,
			wantTasks:    nil,
			wantErr:      errNoSkeletonFound,
		},
		// no main skeleton found
		{
			peers: []*syncPeer{
				{peer: &P2PPeer{id: "peer1", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 1000, irreversibleHeight: 1000},
				{peer: &P2PPeer{id: "peer2", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 800, irreversibleHeight: 800},
			},
			mainSyncPeer: "peer1",
			testType:     4,
			wantTasks:    nil,
			wantErr:      errNoMainSkeleton,
		},
		// skeleton length mismatch
		{
			peers: []*syncPeer{
				{peer: &P2PPeer{id: "peer1", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 1000, irreversibleHeight: 1000},
				{peer: &P2PPeer{id: "peer2", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 800, irreversibleHeight: 800},
			},
			mainSyncPeer: "peer1",
			testType:     5,
			wantTasks: []*fetchBlocksWork{
				{&chainX[1000].BlockHeader, &chainX[1100].BlockHeader}, {&chainX[1100].BlockHeader, &chainX[1200].BlockHeader},
				{&chainX[1200].BlockHeader, &chainX[1300].BlockHeader}, {&chainX[1300].BlockHeader, &chainX[1400].BlockHeader},
				{&chainX[1400].BlockHeader, &chainX[1500].BlockHeader}, {&chainX[1500].BlockHeader, &chainX[1600].BlockHeader},
				{&chainX[1600].BlockHeader, &chainX[1700].BlockHeader}, {&chainX[1700].BlockHeader, &chainX[1800].BlockHeader},
			},
			wantErr: nil,
		},
	}

	for i, c := range cases {
		peers := peers.NewPeerSet(NewPeerSet())
		for _, syncPeer := range c.peers {
			peers.AddPeer(syncPeer.peer)
			peers.SetStatus(syncPeer.peer.id, syncPeer.bestHeight, nil)
			peers.SetJustifiedStatus(syncPeer.peer.id, syncPeer.irreversibleHeight, nil)
		}
		mockChain := mock.NewChain()
		fs := newFastSync(mockChain, &mockFetcher{baseChain: baseChain, peerStatus: peerStatus, testType: c.testType}, nil, peers)
		fs.mainSyncPeer = fs.peers.GetPeer(c.mainSyncPeer)
		tasks, err := fs.createFetchBlocksTasks(baseChain[700])
		if err != c.wantErr {
			t.Errorf("case %d: got %v want %v", i, err, c.wantErr)
		}
		if !reflect.DeepEqual(tasks, c.wantTasks) {
			t.Errorf("case %d: got %v want %v", i, tasks, c.wantTasks)
		}
	}
}
