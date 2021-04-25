package chainmgr

import (
	"errors"
	"math/rand"
	"net"
	"time"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmlibs/flowrate"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/test/mock"
)

type P2PPeer struct {
	id   string
	ip   *net.IPAddr
	flag consensus.ServiceFlag

	srcPeer    *P2PPeer
	remoteNode *Manager
	msgCh      chan []byte
	async      bool
}

func NewP2PPeer(addr, id string, flag consensus.ServiceFlag) *P2PPeer {
	return &P2PPeer{
		id:    id,
		ip:    &net.IPAddr{IP: net.ParseIP(addr)},
		flag:  flag,
		msgCh: make(chan []byte),
		async: false,
	}
}

func (p *P2PPeer) Addr() net.Addr {
	return p.ip
}

func (p *P2PPeer) ID() string {
	return p.id
}

func (p *P2PPeer) IsLAN() bool {
	return false
}

func (p *P2PPeer) Moniker() string {
	return ""
}

func (p *P2PPeer) RemoteAddrHost() string {
	return ""
}

func (p *P2PPeer) ServiceFlag() consensus.ServiceFlag {
	return p.flag
}

func (p *P2PPeer) SetConnection(srcPeer *P2PPeer, node *Manager) {
	p.srcPeer = srcPeer
	p.remoteNode = node
}

func (p *P2PPeer) TrafficStatus() (*flowrate.Status, *flowrate.Status) {
	return nil, nil
}

func (p *P2PPeer) TrySend(b byte, msg interface{}) bool {
	msgBytes := wire.BinaryBytes(msg)
	if p.async {
		p.msgCh <- msgBytes
	} else {
		msgType, msg, _ := decodeMessage(msgBytes)
		p.remoteNode.processMsg(p.srcPeer, msgType, msg)
	}
	return true
}

func (p *P2PPeer) setAsync(b bool) {
	p.async = b
}

func (p *P2PPeer) postMan() {
	for msgBytes := range p.msgCh {
		msgType, msg, _ := decodeMessage(msgBytes)
		time.Sleep(10 * time.Millisecond)
		p.remoteNode.processMsg(p.srcPeer, msgType, msg)
	}
}

type PeerSet struct{}

func NewPeerSet() *PeerSet {
	return &PeerSet{}
}

func (ps *PeerSet) IsBanned(ip string, level byte, reason string) bool {
	return false
}

func (ps *PeerSet) StopPeerGracefully(string) {}

type NetWork struct {
	nodes map[*Manager]P2PPeer
}

func NewNetWork() *NetWork {
	return &NetWork{map[*Manager]P2PPeer{}}
}

func (nw *NetWork) Register(node *Manager, addr, id string, flag consensus.ServiceFlag) {
	peer := NewP2PPeer(addr, id, flag)
	nw.nodes[node] = *peer
}

func (nw *NetWork) HandsShake(nodeA, nodeB *Manager) (*P2PPeer, *P2PPeer, error) {
	B2A, ok := nw.nodes[nodeA]
	if !ok {
		return nil, nil, errors.New("can't find nodeA's p2p peer on network")
	}
	A2B, ok := nw.nodes[nodeB]
	if !ok {
		return nil, nil, errors.New("can't find nodeB's p2p peer on network")
	}

	A2B.SetConnection(&B2A, nodeB)
	B2A.SetConnection(&A2B, nodeA)

	nodeA.AddPeer(&A2B)
	nodeB.AddPeer(&B2A)
	nodeA.SendStatus(B2A.srcPeer)
	nodeB.SendStatus(A2B.srcPeer)
	A2B.setAsync(true)
	B2A.setAsync(true)
	return &B2A, &A2B, nil
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

func mockErrorBlocks(startBlock *types.Block, height uint64, errBlockHeight uint64) []*types.Block {
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
		if block.Height == errBlockHeight {
			block.TransactionsMerkleRoot = bc.NewHash([32]byte{0x1})
		}
		blocks = append(blocks, block)
		indexBlock = block
	}
	return blocks
}

func mockSync(blocks []*types.Block, mempool *mock.Mempool, fastSyncDB dbm.DB) *Manager {
	chain := mock.NewChain()
	peers := peers.NewPeerSet(NewPeerSet())
	chain.SetBestBlockHeader(&blocks[len(blocks)-1].BlockHeader)
	for _, block := range blocks {
		chain.SetBlockByHeight(block.Height, block)
	}

	return &Manager{
		chain:           chain,
		blockKeeper:     newBlockKeeper(chain, peers, fastSyncDB),
		peers:           peers,
		mempool:         mempool,
		txSyncCh:        make(chan *txSyncMsg),
		eventDispatcher: event.NewDispatcher(),
	}
}

func mockTxs(txCount int) ([]*types.Tx, []*bc.Tx) {
	var txs []*types.Tx
	var bcTxs []*bc.Tx
	trueProg := mockControlProgram(60)
	assetID := bc.AssetID{V0: 9999}
	for i := uint64(0); i < uint64(txCount); i++ {
		tx := types.NewTx(types.TxData{
			Version: 1,
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.Hash{V0: i + 1}, assetID, i, i, trueProg, nil)},
			Outputs: []*types.TxOutput{types.NewOriginalTxOutput(assetID, 1, trueProg, nil)},
		})
		txs = append(txs, tx)
		bcTxs = append(bcTxs, tx.Tx)
	}
	return txs, bcTxs
}

func mockControlProgram(length int) []byte {
	var cp []byte
	for i := 0; i < length; i++ {
		cp = append(cp, byte(rand.Intn(1<<8)))
	}
	return cp
}
