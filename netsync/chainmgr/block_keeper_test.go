package chainmgr

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/bytom/bytom/consensus"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/errors"
	msgs "github.com/bytom/bytom/netsync/messages"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/test/mock"
	"github.com/bytom/bytom/testcontrol"
	"github.com/bytom/bytom/testutil"
)

func TestCheckSyncType(t *testing.T) {
	tmp, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatalf("failed to create temporary data folder: %v", err)
	}
	fastSyncDB := dbm.NewDB("testdb", "leveldb", tmp)
	defer func() {
		fastSyncDB.Close()
		os.RemoveAll(tmp)
	}()

	blocks := mockBlocks(nil, 50)
	chain := mock.NewChain()
	chain.SetBestBlockHeader(&blocks[len(blocks)-1].BlockHeader)
	for _, block := range blocks {
		chain.SetBlockByHeight(block.Height, block)
	}

	type syncPeer struct {
		peer               *P2PPeer
		bestHeight         uint64
		irreversibleHeight uint64
	}

	cases := []struct {
		peers    []*syncPeer
		syncType int
	}{
		{
			peers:    []*syncPeer{},
			syncType: noNeedSync,
		},
		{
			peers: []*syncPeer{
				{peer: &P2PPeer{id: "peer1", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 1000, irreversibleHeight: 500},
				{peer: &P2PPeer{id: "peer2", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 50, irreversibleHeight: 50},
			},
			syncType: fastSyncType,
		},
		{
			peers: []*syncPeer{
				{peer: &P2PPeer{id: "peer1", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 1000, irreversibleHeight: 100},
				{peer: &P2PPeer{id: "peer2", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 500, irreversibleHeight: 50},
			},
			syncType: regularSyncType,
		},
		{
			peers: []*syncPeer{
				{peer: &P2PPeer{id: "peer1", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 51, irreversibleHeight: 50},
			},
			syncType: regularSyncType,
		},
		{
			peers: []*syncPeer{
				{peer: &P2PPeer{id: "peer1", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 30, irreversibleHeight: 30},
			},
			syncType: noNeedSync,
		},
		{
			peers: []*syncPeer{
				{peer: &P2PPeer{id: "peer1", flag: consensus.SFFullNode}, bestHeight: 1000, irreversibleHeight: 1000},
			},
			syncType: regularSyncType,
		},
		{
			peers: []*syncPeer{
				{peer: &P2PPeer{id: "peer1", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 1000, irreversibleHeight: 50},
				{peer: &P2PPeer{id: "peer2", flag: consensus.SFFullNode | consensus.SFFastSync}, bestHeight: 800, irreversibleHeight: 800},
			},
			syncType: fastSyncType,
		},
	}

	for i, c := range cases {
		peers := peers.NewPeerSet(NewPeerSet())
		blockKeeper := newBlockKeeper(chain, peers, fastSyncDB)
		for _, syncPeer := range c.peers {
			blockKeeper.peers.AddPeer(syncPeer.peer)
			blockKeeper.peers.SetStatus(syncPeer.peer.id, syncPeer.bestHeight, nil)
			blockKeeper.peers.SetJustifiedStatus(syncPeer.peer.id, syncPeer.irreversibleHeight, nil)
		}
		gotType := blockKeeper.checkSyncType()
		if c.syncType != gotType {
			t.Errorf("case %d: got %d want %d", i, gotType, c.syncType)
		}
	}
}

func TestRegularBlockSync(t *testing.T) {
	if testcontrol.IgnoreTestTemporary {
		return
	}

	baseChain := mockBlocks(nil, 50)
	chainX := append(baseChain, mockBlocks(baseChain[50], 60)...)
	chainY := append(baseChain, mockBlocks(baseChain[50], 70)...)
	chainZ := append(baseChain, mockBlocks(baseChain[50], 200)...)
	chainE := append(baseChain, mockErrorBlocks(baseChain[50], 200, 60)...)

	cases := []struct {
		syncTimeout time.Duration
		aBlocks     []*types.Block
		bBlocks     []*types.Block
		want        []*types.Block
		err         error
	}{
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:20],
			bBlocks:     baseChain[:50],
			want:        baseChain[:50],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     chainX,
			bBlocks:     chainY,
			want:        chainY,
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     chainX[:52],
			bBlocks:     chainY[:53],
			want:        chainY[:53],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     chainX[:52],
			bBlocks:     chainZ,
			want:        chainZ[:180],
			err:         nil,
		},
		{
			syncTimeout: 0 * time.Second,
			aBlocks:     chainX[:52],
			bBlocks:     chainZ,
			want:        chainX[:52],
			err:         errRequestTimeout,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     chainX[:52],
			bBlocks:     chainE,
			want:        chainE[:60],
			err:         protocol.ErrBadStateRoot,
		},
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

	for i, c := range cases {
		a := mockSync(c.aBlocks, nil, testDBA)
		b := mockSync(c.bBlocks, nil, testDBB)
		netWork := NewNetWork()
		netWork.Register(a, "192.168.0.1", "test node A", consensus.SFFullNode)
		netWork.Register(b, "192.168.0.2", "test node B", consensus.SFFullNode)
		if B2A, A2B, err := netWork.HandsShake(a, b); err != nil {
			t.Errorf("fail on peer hands shake %v", err)
		} else {
			go B2A.postMan()
			go A2B.postMan()
		}

		requireBlockTimeout = c.syncTimeout
		a.blockKeeper.syncPeer = a.peers.GetPeer("test node B")
		if err := a.blockKeeper.regularBlockSync(); errors.Root(err) != c.err {
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

func TestRequireBlock(t *testing.T) {
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
		testDBB.Close()
		testDBA.Close()
		os.RemoveAll(tmp)
	}()

	blocks := mockBlocks(nil, 5)
	a := mockSync(blocks[:1], nil, testDBA)
	b := mockSync(blocks[:5], nil, testDBB)
	netWork := NewNetWork()
	netWork.Register(a, "192.168.0.1", "test node A", consensus.SFFullNode)
	netWork.Register(b, "192.168.0.2", "test node B", consensus.SFFullNode)
	if B2A, A2B, err := netWork.HandsShake(a, b); err != nil {
		t.Errorf("fail on peer hands shake %v", err)
	} else {
		go B2A.postMan()
		go A2B.postMan()
	}

	a.blockKeeper.syncPeer = a.peers.GetPeer("test node B")
	b.blockKeeper.syncPeer = b.peers.GetPeer("test node A")
	cases := []struct {
		syncTimeout   time.Duration
		testNode      *Manager
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

	defer func() {
		requireBlockTimeout = 20 * time.Second
	}()

	for i, c := range cases {
		requireBlockTimeout = c.syncTimeout
		got, err := c.testNode.blockKeeper.msgFetcher.requireBlock(c.testNode.blockKeeper.syncPeer.ID(), c.requireHeight)
		if !testutil.DeepEqual(got, c.want) {
			t.Errorf("case %d: got %v want %v", i, got, c.want)
		}
		if errors.Root(err) != c.err {
			t.Errorf("case %d: got %v want %v", i, err, c.err)
		}
	}
}

func TestSendMerkleBlock(t *testing.T) {
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

	cases := []struct {
		txCount        int
		relatedTxIndex []int
	}{
		{
			txCount:        10,
			relatedTxIndex: []int{0, 2, 5},
		},
		{
			txCount:        0,
			relatedTxIndex: []int{},
		},
		{
			txCount:        10,
			relatedTxIndex: []int{},
		},
		{
			txCount:        5,
			relatedTxIndex: []int{0, 1, 2, 3, 4},
		},
		{
			txCount:        20,
			relatedTxIndex: []int{1, 6, 3, 9, 10, 19},
		},
	}

	for _, c := range cases {
		blocks := mockBlocks(nil, 2)
		targetBlock := blocks[1]
		txs, bcTxs := mockTxs(c.txCount)
		var err error

		targetBlock.Transactions = txs
		if targetBlock.TransactionsMerkleRoot, err = types.TxMerkleRoot(bcTxs); err != nil {
			t.Fatal(err)
		}

		spvNode := mockSync(blocks, nil, testDBA)
		fullNode := mockSync(blocks, nil, testDBB)
		netWork := NewNetWork()
		netWork.Register(spvNode, "192.168.0.1", "spv_node", consensus.SFFastSync)
		netWork.Register(fullNode, "192.168.0.2", "full_node", consensus.DefaultServices)

		var F2S *P2PPeer
		if F2S, _, err = netWork.HandsShake(spvNode, fullNode); err != nil {
			t.Errorf("fail on peer hands shake %v", err)
		}

		completed := make(chan error)
		go func() {
			msgBytes := <-F2S.msgCh
			_, msg, _ := decodeMessage(msgBytes)
			switch m := msg.(type) {
			case *msgs.MerkleBlockMessage:
				var relatedTxIDs []*bc.Hash
				for _, rawTx := range m.RawTxDatas {
					tx := &types.Tx{}
					if err := tx.UnmarshalText(rawTx); err != nil {
						completed <- err
					}

					relatedTxIDs = append(relatedTxIDs, &tx.ID)
				}
				var txHashes []*bc.Hash
				for _, hashByte := range m.TxHashes {
					hash := bc.NewHash(hashByte)
					txHashes = append(txHashes, &hash)
				}
				if ok := types.ValidateTxMerkleTreeProof(txHashes, m.Flags, relatedTxIDs, targetBlock.TransactionsMerkleRoot); !ok {
					completed <- errors.New("validate tx fail")
				}
				completed <- nil
			}
		}()

		spvPeer := fullNode.peers.GetPeer("spv_node")
		for i := 0; i < len(c.relatedTxIndex); i++ {
			spvPeer.AddFilterAddress(txs[c.relatedTxIndex[i]].Outputs[0].ControlProgram)
		}
		msg := &msgs.GetMerkleBlockMessage{RawHash: targetBlock.Hash().Byte32()}
		fullNode.handleGetMerkleBlockMsg(spvPeer, msg)
		if err := <-completed; err != nil {
			t.Fatal(err)
		}
	}
}

func TestLocateBlocks(t *testing.T) {
	if testcontrol.IgnoreTestTemporary {
		return
	}

	maxNumOfBlocksPerMsg = 5
	blocks := mockBlocks(nil, 100)
	cases := []struct {
		locator    []uint64
		stopHash   bc.Hash
		wantHeight []uint64
		wantErr    error
	}{
		{
			locator:    []uint64{20},
			stopHash:   blocks[100].Hash(),
			wantHeight: []uint64{20, 21, 22, 23, 24},
			wantErr:    nil,
		},
		{
			locator:    []uint64{20},
			stopHash:   bc.NewHash([32]byte{0x01, 0x02}),
			wantHeight: []uint64{},
			wantErr:    mock.ErrFoundHeaderByHash,
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

		mockTimeout := func() bool { return false }
		got, err := bk.locateBlocks(locator, &c.stopHash, mockTimeout)
		if err != c.wantErr {
			t.Errorf("case %d: got %v want err = %v", i, err, c.wantErr)
		}

		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func TestLocateHeaders(t *testing.T) {
	if testcontrol.IgnoreTestTemporary {
		return
	}

	defer func() {
		maxNumOfHeadersPerMsg = 1000
	}()
	maxNumOfHeadersPerMsg = 10
	blocks := mockBlocks(nil, 150)
	blocksHash := []bc.Hash{}
	for _, block := range blocks {
		blocksHash = append(blocksHash, block.Hash())
	}

	cases := []struct {
		chainHeight uint64
		locator     []uint64
		stopHash    *bc.Hash
		skip        uint64
		wantHeight  []uint64
		err         error
	}{
		{
			chainHeight: 100,
			locator:     []uint64{90},
			stopHash:    &blocksHash[100],
			skip:        0,
			wantHeight:  []uint64{90, 91, 92, 93, 94, 95, 96, 97, 98, 99},
			err:         nil,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    &blocksHash[24],
			skip:        0,
			wantHeight:  []uint64{20, 21, 22, 23, 24},
			err:         nil,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    &blocksHash[20],
			wantHeight:  []uint64{20},
			err:         nil,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    &blocksHash[120],
			wantHeight:  []uint64{},
			err:         mock.ErrFoundHeaderByHash,
		},
		{
			chainHeight: 100,
			locator:     []uint64{120, 70},
			stopHash:    &blocksHash[78],
			wantHeight:  []uint64{70, 71, 72, 73, 74, 75, 76, 77, 78},
			err:         nil,
		},
		{
			chainHeight: 100,
			locator:     []uint64{15},
			stopHash:    &blocksHash[10],
			skip:        10,
			wantHeight:  []uint64{},
			err:         nil,
		},
		{
			chainHeight: 100,
			locator:     []uint64{15},
			stopHash:    &blocksHash[80],
			skip:        10,
			wantHeight:  []uint64{15, 26, 37, 48, 59, 70, 80},
			err:         nil,
		},
		{
			chainHeight: 100,
			locator:     []uint64{0},
			stopHash:    &blocksHash[100],
			skip:        9,
			wantHeight:  []uint64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90},
			err:         nil,
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

		got, err := bk.locateHeaders(locator, c.stopHash, c.skip, maxNumOfHeadersPerMsg)
		if err != c.err {
			t.Errorf("case %d: got %v want err = %v", i, err, c.err)
		}
		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}
