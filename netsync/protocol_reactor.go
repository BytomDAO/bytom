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
	"github.com/bytom/p2p/connection"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	protocolHandshakeTimeout = time.Second * 10
	handshakeRetryTicker     = 4 * time.Second
)

var (
	//ErrProtocolHandshakeTimeout peers handshake timeout
	ErrProtocolHandshakeTimeout = errors.New("Protocol handshake timeout")
	errStatusRequest            = errors.New("Status request error")
	errDiffGenesisHash          = errors.New("Different genesis hash")
)

type initalPeerStatus struct {
	peerID      string
	height      uint64
	hash        *bc.Hash
	genesisHash *bc.Hash
}

//ProtocolReactor handles new coming protocol message.
type ProtocolReactor struct {
	p2p.BaseReactor

	sm          *SyncManager
	chain       *protocol.Chain
	blockKeeper *blockKeeper
	txPool      *protocol.TxPool
	peers       *peerSet
	handshakeMu sync.Mutex
	genesisHash bc.Hash

	newPeerCh    chan struct{}
	txSyncCh     chan *txsync
	peerStatusCh chan *initalPeerStatus
}

// NewProtocolReactor returns the reactor of whole blockchain.
func NewProtocolReactor(sm *SyncManager, chain *protocol.Chain, txPool *protocol.TxPool, blockPeer *blockKeeper, peers *peerSet, newPeerCh chan struct{}, txSyncCh chan *txsync) *ProtocolReactor {
	pr := &ProtocolReactor{
		sm:           sm,
		chain:        chain,
		blockKeeper:  blockPeer,
		txPool:       txPool,
		peers:        peers,
		newPeerCh:    newPeerCh,
		txSyncCh:     txSyncCh,
		peerStatusCh: make(chan *initalPeerStatus),
	}
	pr.BaseReactor = *p2p.NewBaseReactor("ProtocolReactor", pr)
	genesisBlock, _ := pr.chain.GetBlockByHeight(0)
	pr.genesisHash = genesisBlock.Hash()

	return pr
}

// GetChannels implements Reactor
func (pr *ProtocolReactor) GetChannels() []*connection.ChannelDescriptor {
	return []*connection.ChannelDescriptor{
		&connection.ChannelDescriptor{
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
	if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{&StatusRequestMessage{}}); !ok {
		return errStatusRequest
	}
	retryTicker := time.Tick(handshakeRetryTicker)
	handshakeWait := time.NewTimer(protocolHandshakeTimeout)
	for {
		select {
		case status := <-pr.peerStatusCh:
			if status.peerID == peer.Key {
				if strings.Compare(pr.genesisHash.String(), status.genesisHash.String()) != 0 {
					log.Info("Remote peer genesis block hash:", status.genesisHash.String(), " local hash:", pr.genesisHash.String())
					return errDiffGenesisHash
				}
				pr.blockKeeper.peers.addPeer(peer, status.height, status.hash)
				pr.syncTransactions(pr.blockKeeper.peers.getPeer(peer.Key))
				pr.newPeerCh <- struct{}{}
				return nil
			}
		case <-retryTicker:
			if peer == nil {
				return errPeerDropped
			}
			if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{&StatusRequestMessage{}}); !ok {
				return errStatusRequest
			}
		case <-handshakeWait.C:
			return ErrProtocolHandshakeTimeout
		}
	}
}

// RemovePeer implements Reactor by removing peer from the pool.
func (pr *ProtocolReactor) RemovePeer(peer *p2p.Peer, reason interface{}) {
	pr.peers.removePeer(peer.Key)
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (pr *ProtocolReactor) Receive(chID byte, src *p2p.Peer, msgBytes []byte) {
	_, msg, err := DecodeMessage(msgBytes)
	if err != nil {
		log.Errorf("Error decoding message %v", err)
		return
	}

	peer := pr.peers.getPeer(src.Key)
	if peer == nil {
		return
	}

	switch msg := msg.(type) {
	case *GetBlockMessage:
		pr.sm.handleGetBlockMsg(peer, msg)

	case *BlockMessage:
		pr.blockKeeper.AddBlock(msg.GetBlock(), src.Key)

	case *StatusRequestMessage:
		pr.sm.handleStatusRequestMsg(peer)

	case *StatusResponseMessage:
		peerStatus := &initalPeerStatus{
			peerID:      src.Key,
			height:      msg.Height,
			hash:        msg.GetHash(),
			genesisHash: msg.GetGenesisHash(),
		}
		pr.peerStatusCh <- peerStatus

	case *TransactionMessage:
		pr.sm.handleTransactionMsg(peer, msg)

	case *MineBlockMessage:
		pr.sm.handleMineBlockMsg(peer, msg)

	case *GetHeadersMessage:
		pr.sm.handleGetHeadersMsg(peer, msg)

	case *HeadersMessage:
		headers, err := msg.GetHeaders()
		if err != nil {
			return
		}
		hmsg := &headersMsg{headers: headers, peerID: src.Key}
		pr.blockKeeper.headersProcessCh <- hmsg

	case *GetBlocksMessage:
		pr.sm.handleGetBlocksMsg(peer, msg)

	case *BlocksMessage:
		peer := pr.blockKeeper.peers.getPeer(src.Key)
		blocks, _ := msg.GetBlocks()
		if peer != nil {
			pr.blockKeeper.blocksProcessCh <- &blocksMsg{blocks: blocks, peerID: peer.ID()}
		}

	default:
		log.Error(cmn.Fmt("Unknown message type %v", reflect.TypeOf(msg)))
	}
}
