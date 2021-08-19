package consensusmgr

import (
	"math/rand"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/tendermint/tmlibs/flowrate"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/p2p"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/casper"
)

type p2peer struct {
}

func (p *p2peer) Addr() net.Addr {
	return nil
}

func (p *p2peer) ID() string {
	return ""
}

func (p *p2peer) Moniker() string {
	return ""
}

func (p *p2peer) RemoteAddrHost() string {
	return ""
}
func (p *p2peer) ServiceFlag() consensus.ServiceFlag {
	return 0
}
func (p *p2peer) TrafficStatus() (*flowrate.Status, *flowrate.Status) {
	return nil, nil
}
func (p *p2peer) TrySend(byte, interface{}) bool {
	return true
}
func (p *p2peer) IsLAN() bool {
	return false
}

func mockBlocks(startBlock *types.Block, height uint64) []*types.Block {
	blocks := []*types.Block{}
	indexBlock := &types.Block{}
	if startBlock == nil {
		indexBlock = &types.Block{BlockHeader: types.BlockHeader{Version: uint64(rand.Uint32())}}
		blocks = append(blocks, indexBlock)
	} else {
		indexBlock = startBlock
	}

	for indexBlock.Height < height {
		block := &types.Block{
			BlockHeader: types.BlockHeader{
				Height:            indexBlock.Height + 1,
				PreviousBlockHash: indexBlock.Hash(),
				Version:           uint64(rand.Uint32()),
			},
		}
		blocks = append(blocks, block)
		indexBlock = block
	}
	return blocks
}

type mockSW struct {
}

func (s *mockSW) AddReactor(name string, reactor p2p.Reactor) p2p.Reactor {
	return nil
}

type mockChain struct {
}

func (c *mockChain) BestBlockHeight() uint64 {
	return 0
}

func (c *mockChain) GetHeaderByHash(*bc.Hash) (*types.BlockHeader, error) {
	return nil, nil
}

func (c *mockChain) ProcessBlock(*types.Block) (bool, error) {
	return false, nil
}

func (c *mockChain) ProcessBlockVerification(*casper.ValidCasperSignMsg) error {
	return nil
}

type mockPeers struct {
	msgCount       *int
	knownBlock     *bc.Hash
	blockHeight    *uint64
	knownSignature *[]byte
}

func newMockPeers(msgCount *int, knownBlock *bc.Hash, blockHeight *uint64, signature *[]byte) *mockPeers {
	return &mockPeers{
		msgCount:       msgCount,
		knownBlock:     knownBlock,
		blockHeight:    blockHeight,
		knownSignature: signature,
	}
}

func (ps *mockPeers) AddPeer(peer peers.BasePeer) {

}

func (ps *mockPeers) BroadcastMsg(bm peers.BroadcastMsg) error {
	*ps.msgCount++
	return nil
}
func (ps *mockPeers) GetPeer(id string) *peers.Peer {
	return &peers.Peer{BasePeer: &p2peer{}}
}
func (ps *mockPeers) MarkBlock(peerID string, hash *bc.Hash) {
	*ps.knownBlock = *hash
}

func (ps *mockPeers) MarkBlockVerification(peerID string, signature []byte) {
	*ps.knownSignature = append(*ps.knownSignature, signature...)
}

func (ps *mockPeers) ProcessIllegal(peerID string, level byte, reason string) {

}
func (p *mockPeers) RemovePeer(peerID string) {

}
func (ps *mockPeers) SetStatus(peerID string, height uint64, hash *bc.Hash) {
	*ps.blockHeight = height
}

func TestBlockProposeMsgBroadcastLoop(t *testing.T) {
	dispatcher := event.NewDispatcher()
	msgCount := 0
	blockHeight := 100
	mgr := NewManager(&mockSW{}, &mockChain{}, newMockPeers(&msgCount, nil, nil, nil), dispatcher)
	blocks := mockBlocks(nil, uint64(blockHeight))

	mgr.Start()
	defer mgr.Stop()
	time.Sleep(10 * time.Millisecond)
	for _, block := range blocks {
		mgr.eventDispatcher.Post(event.NewProposedBlockEvent{Block: *block})
	}
	time.Sleep(10 * time.Millisecond)
	if msgCount != blockHeight+1 {
		t.Fatalf("broad propose block msg err. got:%d\n want:%d", msgCount, blockHeight+1)
	}
}

func TestBlockVerificationMsgBroadcastLoop(t *testing.T) {
	dispatcher := event.NewDispatcher()
	msgCount := 0
	blockHeight := 100
	mgr := NewManager(&mockSW{}, &mockChain{}, newMockPeers(&msgCount, nil, nil, nil), dispatcher)
	blocks := mockBlocks(nil, uint64(blockHeight))

	mgr.Start()
	defer mgr.Stop()
	time.Sleep(10 * time.Millisecond)
	for _, block := range blocks {
		mgr.eventDispatcher.Post(casper.ValidCasperSignMsg{TargetHash: block.Hash(), Signature: []byte{0x1, 0x2}, PubKey: "011022"})
	}
	time.Sleep(10 * time.Millisecond)
	if msgCount != blockHeight+1 {
		t.Fatalf("broad propose block msg err. got:%d\n want:%d", msgCount, blockHeight+1)
	}
}

func TestProcessBlockProposeMsg(t *testing.T) {
	dispatcher := event.NewDispatcher()
	msgCount := 0
	var knownBlock bc.Hash
	blockHeight := uint64(0)
	peerID := "Peer1"
	mgr := NewManager(&mockSW{}, &mockChain{}, newMockPeers(&msgCount, &knownBlock, &blockHeight, nil), dispatcher)
	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Height:            100,
			PreviousBlockHash: bc.NewHash([32]byte{0x1}),
			Version:           uint64(rand.Uint32()),
		},
	}
	msg, err := NewBlockProposeMsg(block)
	if err != nil {
		t.Fatal("create new block propose msg err", err)
	}

	mgr.processMsg(peerID, 0, msg)
	if knownBlock != block.Hash() {
		t.Fatalf("mark propose block msg err. got:%d\n want:%d", knownBlock, block.Hash())
	}

	if blockHeight != block.Height {
		t.Fatalf("set peer status err. got:%d\n want:%d", blockHeight, block.Height)
	}
}

func TestProcessBlockVerificationMsg(t *testing.T) {
	dispatcher := event.NewDispatcher()
	msgCount := 0
	knownSignature := []byte{}
	peerID := "Peer1"
	mgr := NewManager(&mockSW{}, &mockChain{}, newMockPeers(&msgCount, nil, nil, &knownSignature), dispatcher)
	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Height:            100,
			PreviousBlockHash: bc.NewHash([32]byte{0x1}),
			Version:           uint64(rand.Uint32()),
		},
	}

	signature := []byte{0x01, 0x02}
	msg := NewBlockVerificationMsg(block.Hash(), block.Hash(), []byte{0x03, 0x04}, signature)

	mgr.processMsg(peerID, 0, msg)

	if !reflect.DeepEqual(knownSignature, signature) {
		t.Fatalf("set peer status err. got:%d\n want:%d", knownSignature, signature)
	}
}
