package netsync

import (
	"reflect"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/bytom/errors"
	"github.com/bytom/p2p"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	// BlockchainChannel is a channel for blocks and status updates
	BlockchainChannel        = byte(0x40)
	protocolHandshakeTimeout = time.Second * 10
	handshakeRetryTicker     = 4 * time.Second
)

var (
	//ErrProtocolHandshakeTimeout peers handshake timeout
	ErrProtocolHandshakeTimeout = errors.New("Protocol handshake timeout")
	ErrStatusRequest            = errors.New("Status request error")
	ErrDiffGenesisHash          = errors.New("Different genesis hash")
)

// Response describes the response standard.
type Response struct {
	Status string      `json:"status,omitempty"`
	Msg    string      `json:"msg,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

type initalPeerStatus struct {
	peerID      string
	height      uint64
	hash        *bc.Hash
	genesisHash *bc.Hash
}

//ProtocolReactor handles new coming protocol message.
type ProtocolReactor struct {
	p2p.BaseReactor

	chain       *protocol.Chain
	blockKeeper *blockKeeper
	txPool      *protocol.TxPool
	sw          *p2p.Switch
	fetcher     *Fetcher
	peers       *peerSet
	handshakeMu sync.Mutex
	genesisHash bc.Hash

	newPeerCh      chan struct{}
	quitReqBlockCh chan *string
	txSyncCh       chan *txsync
	peerStatusCh   chan *initalPeerStatus
}

// NewProtocolReactor returns the reactor of whole blockchain.
func NewProtocolReactor(chain *protocol.Chain, txPool *protocol.TxPool, sw *p2p.Switch, blockPeer *blockKeeper, fetcher *Fetcher, peers *peerSet, newPeerCh chan struct{}, txSyncCh chan *txsync, quitReqBlockCh chan *string) *ProtocolReactor {
	pr := &ProtocolReactor{
		chain:          chain,
		blockKeeper:    blockPeer,
		txPool:         txPool,
		sw:             sw,
		fetcher:        fetcher,
		peers:          peers,
		newPeerCh:      newPeerCh,
		txSyncCh:       txSyncCh,
		quitReqBlockCh: quitReqBlockCh,
		peerStatusCh:   make(chan *initalPeerStatus),
	}
	pr.BaseReactor = *p2p.NewBaseReactor("ProtocolReactor", pr)
	genesisBlock, _ := pr.chain.GetBlockByHeight(0)
	pr.genesisHash = genesisBlock.Hash()

	return pr
}

// GetChannels implements Reactor
func (pr *ProtocolReactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		&p2p.ChannelDescriptor{
			ID:                BlockchainChannel,
			Priority:          5,
			SendQueueCapacity: 100,
		},
	}
}

// OnStart implements BaseService
func (pr *ProtocolReactor) OnStart() error {
	pr.BaseReactor.OnStart()
	return nil
}

// OnStop implements BaseService
func (pr *ProtocolReactor) OnStop() {
	pr.BaseReactor.OnStop()
}

// syncTransactions starts sending all currently pending transactions to the given peer.
func (pr *ProtocolReactor) syncTransactions(p *peer) {
	if p == nil {
		return
	}
	pending := pr.txPool.GetTransactions()
	if len(pending) == 0 {
		return
	}
	txs := make([]*types.Tx, len(pending))
	for i, batch := range pending {
		txs[i] = batch.Tx
	}
	pr.txSyncCh <- &txsync{p, txs}
}

// AddPeer implements Reactor by sending our state to peer.
func (pr *ProtocolReactor) AddPeer(peer *p2p.Peer) error {
	pr.handshakeMu.Lock()
	defer pr.handshakeMu.Unlock()
	if peer == nil {
		return errPeerDropped
	}
	if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{&StatusRequestMessage{}}); !ok {
		return ErrStatusRequest
	}
	retryTicker := time.Tick(handshakeRetryTicker)
	handshakeWait := time.NewTimer(protocolHandshakeTimeout)
	for {
		select {
		case status := <-pr.peerStatusCh:
			if status.peerID == peer.Key {
				if strings.Compare(pr.genesisHash.String(), status.genesisHash.String()) != 0 {
					log.Info("Remote peer genesis block hash:", status.genesisHash.String(), " local hash:", pr.genesisHash.String())
					return ErrDiffGenesisHash
				}
				pr.peers.AddPeer(peer)
				pr.peers.SetPeerStatus(status.peerID, status.height, status.hash)
				prPeer, ok := pr.peers.Peer(peer.Key)
				if !ok {
					return errPeerDropped
				}
				pr.syncTransactions(prPeer)
				pr.newPeerCh <- struct{}{}
				return nil
			}
		case <-retryTicker:
			if peer == nil {
				return errPeerDropped
			}
			if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{&StatusRequestMessage{}}); !ok {
				return ErrStatusRequest
			}
		case <-handshakeWait.C:
			return ErrProtocolHandshakeTimeout
		}
	}
}

// RemovePeer implements Reactor by removing peer from the pool.
func (pr *ProtocolReactor) RemovePeer(peer *p2p.Peer, reason interface{}) {
	select {
	case pr.quitReqBlockCh <- &peer.Key:
	default:
		log.Warning("quitReqBlockCh is full")
	}
	pr.peers.RemovePeer(peer.Key)
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (pr *ProtocolReactor) Receive(chID byte, src *p2p.Peer, msgBytes []byte) {
	_, msg, err := DecodeMessage(msgBytes)
	if err != nil {
		log.Errorf("Error decoding message %v", err)
		return
	}
	log.WithFields(log.Fields{"peerID": src.Key, "msg": msg}).Info("Receive request")

	switch msg := msg.(type) {
	case *BlockRequestMessage:
		var block *types.Block
		var err error
		if msg.Height != 0 {
			block, err = pr.chain.GetBlockByHeight(msg.Height)
		} else {
			block, err = pr.chain.GetBlockByHash(msg.GetHash())
		}
		if err != nil {
			log.Errorf("Fail on BlockRequestMessage get block: %v", err)
			return
		}
		response, err := NewBlockResponseMessage(block)
		if err != nil {
			log.Errorf("Fail on BlockRequestMessage create response: %v", err)
			return
		}
		src.TrySend(BlockchainChannel, struct{ BlockchainMessage }{response})

	case *BlockResponseMessage:
		log.Info("BlockResponseMessage height:", msg.GetBlock().Height)
		pr.blockKeeper.AddBlock(msg.GetBlock(), src.Key)

	case *StatusRequestMessage:
		blockHeader := pr.chain.BestBlockHeader()
		src.TrySend(BlockchainChannel, struct{ BlockchainMessage }{NewStatusResponseMessage(blockHeader, &pr.genesisHash)})

	case *StatusResponseMessage:
		peerStatus := &initalPeerStatus{
			peerID:      src.Key,
			height:      msg.Height,
			hash:        msg.GetHash(),
			genesisHash: msg.GetGenesisHash(),
		}
		pr.peerStatusCh <- peerStatus

	case *TransactionNotifyMessage:
		tx, err := msg.GetTransaction()
		if err != nil {
			log.Errorf("Error decoding new tx %v", err)
			return
		}
		pr.blockKeeper.AddTx(tx, src.Key)

	case *MineBlockMessage:
		block, err := msg.GetMineBlock()
		if err != nil {
			log.Errorf("Error decoding mined block %v", err)
			return
		}
		// Mark the peer as owning the block and schedule it for import
		hash := block.Hash()
		pr.peers.MarkBlock(src.Key, &hash)
		pr.fetcher.Enqueue(src.Key, block)
		pr.peers.SetPeerStatus(src.Key, block.Height, &hash)

	default:
		log.Error(cmn.Fmt("Unknown message type %v", reflect.TypeOf(msg)))
	}
}
