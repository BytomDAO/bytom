package consensusmgr

import (
	"testing"
	"time"

	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/casper"
)

type peerMgr struct {
}

func (pm *peerMgr) IsBanned(ip string, level byte, reason string) bool {
	return false
}

func (pm *peerMgr) StopPeerGracefully(string) {
	return
}

type chain struct {
	blocks []uint64
}

func newChain() *chain {
	blocks := make([]uint64, 1, 1)
	blocks[0] = 99
	return &chain{
		blocks: blocks,
	}
}

func (c *chain) BestBlockHeight() uint64 {
	return c.blocks[len(c.blocks)-1]
}

func (c *chain) GetHeaderByHash(*bc.Hash) (*types.BlockHeader, error) {
	return nil, nil
}

func (c *chain) ProcessBlock(block *types.Block) (bool, error) {
	c.blocks = append(c.blocks, block.Height)
	return false, nil
}

func (c *chain) ProcessBlockVerification(*casper.ValidCasperSignMsg) error {
	return nil
}

func TestBlockFetcher(t *testing.T) {
	peers := peers.NewPeerSet(&peerMgr{})
	testCase := []struct {
		blockMsg *blockMsg
		height   uint64
	}{
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 100,
					},
				},
			},
			height: 100,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 101,
					},
				},
			},
			height: 101,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 105,
					},
				},
			},
			height: 101,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 200,
					},
				},
			},
			height: 101,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 104,
					},
				},
			},
			height: 101,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 103,
					},
				},
			},
			height: 101,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 102,
					},
				},
			},
			height: 105,
		},
	}
	fetcher := newBlockFetcher(newChain(), peers)
	go fetcher.blockProcessorLoop()
	for i, c := range testCase {
		fetcher.processNewBlock(c.blockMsg)
		time.Sleep(10 * time.Millisecond)
		chainHeight := fetcher.chain.BestBlockHeight()
		if chainHeight != c.height {
			t.Fatalf("test block fetcher error. index %d expected chain height %d but got %d", i, chainHeight, c.height)
		}
	}
}

func TestAddBlockMsg(t *testing.T) {
	peers := peers.NewPeerSet(&peerMgr{})
	testPeer := "peer1"
	testCase := []struct {
		blocksMsg  []*blockMsg
		limit      int
		queueSize  int
		msgSetSize int
		msgCounter int
	}{
		//normal test
		{
			blocksMsg: []*blockMsg{
				{
					block: &types.Block{
						BlockHeader: types.BlockHeader{
							Height: 100,
						},
					},
					peerID: testPeer,
				},
				{
					block: &types.Block{
						BlockHeader: types.BlockHeader{
							Height: 101,
						},
					},
					peerID: testPeer,
				},
				{
					block: &types.Block{
						BlockHeader: types.BlockHeader{
							Height: 102,
						},
					},
					peerID: testPeer,
				},
			},
			limit:      5,
			queueSize:  3,
			msgSetSize: 3,
			msgCounter: 3,
		},
		// test DOS
		{
			blocksMsg: []*blockMsg{
				{
					block: &types.Block{
						BlockHeader: types.BlockHeader{
							Version: 1,
							Height:  100,
						},
					},
					peerID: testPeer,
				},
				{
					block: &types.Block{
						BlockHeader: types.BlockHeader{
							Version: 2,
							Height:  100,
						},
					},
					peerID: testPeer,
				},
				{
					block: &types.Block{
						BlockHeader: types.BlockHeader{
							Version: 3,
							Height:  100,
						},
					},
					peerID: testPeer,
				},
				{
					block: &types.Block{
						BlockHeader: types.BlockHeader{
							Version: 4,
							Height:  100,
						},
					},
					peerID: testPeer,
				},
			},
			limit:      3,
			queueSize:  3,
			msgSetSize: 3,
			msgCounter: 3,
		},

		// test msg height does not meet the requirements
		{
			blocksMsg: []*blockMsg{
				{
					block: &types.Block{
						BlockHeader: types.BlockHeader{
							Version: 1,
							Height:  98,
						},
					},
					peerID: testPeer,
				},
				{
					block: &types.Block{
						BlockHeader: types.BlockHeader{
							Version: 2,
							Height:  97,
						},
					},
					peerID: testPeer,
				},
				{
					block: &types.Block{
						BlockHeader: types.BlockHeader{
							Version: 3,
							Height:  164,
						},
					},
					peerID: testPeer,
				},
				{
					block: &types.Block{
						BlockHeader: types.BlockHeader{
							Version: 4,
							Height:  165,
						},
					},
					peerID: testPeer,
				},
			},
			limit:      5,
			queueSize:  0,
			msgSetSize: 0,
			msgCounter: 0,
		},
	}

	for i, c := range testCase {
		fetcher := newBlockFetcher(newChain(), peers)
		for _, msg := range c.blocksMsg {
			fetcher.add(msg, c.limit)
		}

		if fetcher.queue.Size() != c.queueSize {
			t.Fatalf("index: %d queue size err got %d: want %d", i, fetcher.queue.Size(), c.queueSize)
		}

		if len(fetcher.msgSet) != c.msgSetSize {
			t.Fatalf("index: %d msg set size err got %d: want %d", i, len(fetcher.msgSet), c.msgSetSize)
		}

		if fetcher.msgCounter[testPeer] != c.msgCounter {
			t.Fatalf("index: %d peer msg counter err got %d: want %d", i, fetcher.msgCounter[testPeer], c.msgCounter)
		}
	}
}
