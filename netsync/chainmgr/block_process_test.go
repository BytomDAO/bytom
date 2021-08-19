package chainmgr

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/test/mock"
	"github.com/bytom/bytom/testcontrol"
)

func TestBlockProcess(t *testing.T) {
	if testcontrol.IgnoreTestTemporary {
		return
	}

	tmp, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	testDB := dbm.NewDB("testdb", "leveldb", tmp)
	defer testDB.Close()

	cases := []struct {
		blocks      []*types.Block
		startHeight uint64
		stopHeight  uint64
	}{
		{
			blocks:      mockBlocks(nil, 200),
			startHeight: 100,
			stopHeight:  200,
		},
		{
			blocks:      mockBlocks(nil, 200),
			startHeight: 110,
			stopHeight:  100,
		},
		{
			blocks:      mockErrorBlocks(nil, 200, 150),
			startHeight: 100,
			stopHeight:  149,
		},
	}
	s := newStorage(testDB)
	mockChain := mock.NewChain()
	for i, c := range cases {
		for i := 0; i <= len(c.blocks)/2; i++ {
			mockChain.SetBlockByHeight(uint64(i), c.blocks[i])
			mockChain.SetBestBlockHeader(&c.blocks[i].BlockHeader)
		}

		if err := s.writeBlocks("testPeer", c.blocks); err != nil {
			t.Fatal(err)
		}

		bp := newBlockProcessor(mockChain, s, peers.NewPeerSet(nil))
		downloadNotifyCh := make(chan struct{}, 1)
		ProcessStopCh := make(chan struct{})
		var wg sync.WaitGroup
		go func() {
			time.Sleep(1 * time.Second)
			close(downloadNotifyCh)
		}()
		wg.Add(1)

		bp.process(downloadNotifyCh, ProcessStopCh, c.startHeight, &wg)
		if bp.chain.BestBlockHeight() != c.stopHeight {
			t.Fatalf("TestBlockProcess index: %d fail: got %d want %d", i, bp.chain.BestBlockHeight(), c.stopHeight)
		}
	}
}
