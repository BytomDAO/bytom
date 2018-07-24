package netsync

import (
	"errors"
	"net"

	wire "github.com/tendermint/go-wire"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/test/mock"
)

type P2PPeer struct {
	id   string
	ip   *net.IPAddr
	flag consensus.ServiceFlag

	srcPeer    *P2PPeer
	remoteNode *SyncManager
}

func NewP2PPeer(addr, id string, flag consensus.ServiceFlag) *P2PPeer {
	return &P2PPeer{
		id:   id,
		ip:   &net.IPAddr{IP: net.ParseIP(addr)},
		flag: flag,
	}
}

func (p *P2PPeer) Addr() net.Addr {
	return p.ip
}

func (p *P2PPeer) ID() string {
	return p.id
}

func (p *P2PPeer) ServiceFlag() consensus.ServiceFlag {
	return p.flag
}

func (p *P2PPeer) SetConnection(srcPeer *P2PPeer, node *SyncManager) {
	p.srcPeer = srcPeer
	p.remoteNode = node
}

func (p *P2PPeer) TrySend(b byte, msg interface{}) bool {
	msgBytes := wire.BinaryBytes(msg)
	msgType, msg, _ := DecodeMessage(msgBytes)
	p.remoteNode.processMsg(p.srcPeer, msgType, msg)
	return true
}

type PeerSet struct{}

func NewPeerSet() *PeerSet {
	return &PeerSet{}
}

func (ps *PeerSet) AddBannedPeer(string) error { return nil }
func (ps *PeerSet) StopPeerGracefully(string)  {}

type NetWork struct {
	nodes map[*SyncManager]P2PPeer
}

func NewNetWork() *NetWork {
	return &NetWork{map[*SyncManager]P2PPeer{}}
}

func (nw *NetWork) Register(node *SyncManager, addr, id string, flag consensus.ServiceFlag) {
	peer := NewP2PPeer(addr, id, flag)
	nw.nodes[node] = *peer
}

func (nw *NetWork) HandsShake(nodeA, nodeB *SyncManager) error {
	B2A, ok := nw.nodes[nodeA]
	if !ok {
		return errors.New("can't find nodeA's p2p peer on network")
	}
	A2B, ok := nw.nodes[nodeB]
	if !ok {
		return errors.New("can't find nodeB's p2p peer on network")
	}

	B2A.SetConnection(&A2B, nodeA)
	A2B.SetConnection(&B2A, nodeB)

	nodeA.handleStatusRequestMsg(&A2B)
	nodeB.handleStatusRequestMsg(&B2A)
	return nil
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

func mockSync(blocks []*types.Block) *SyncManager {
	chain := mock.NewChain()
	peers := newPeerSet(NewPeerSet())
	chain.SetBestBlockHeader(&blocks[len(blocks)-1].BlockHeader)
	for _, block := range blocks {
		chain.SetBlockByHeight(block.Height, block)
	}

	genesis, _ := chain.GetHeaderByHeight(0)
	return &SyncManager{
		genesisHash: genesis.Hash(),
		chain:       chain,
		blockKeeper: newBlockKeeper(chain, peers),
		peers:       peers,
	}
}
