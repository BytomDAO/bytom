package chainmgr

import (
	"errors"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"

	cfg "github.com/bytom/bytom/config"
	"github.com/bytom/bytom/consensus"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/event"
	msgs "github.com/bytom/bytom/netsync/messages"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/p2p"
	"github.com/bytom/bytom/p2p/security"
	core "github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

const (
	logModule = "netsync"
)

// Chain is the interface for Bytom core
type Chain interface {
	BestBlockHeader() *types.BlockHeader
	LastJustifiedHeader() (*types.BlockHeader, error)
	BestBlockHeight() uint64
	GetBlockByHash(*bc.Hash) (*types.Block, error)
	GetBlockByHeight(uint64) (*types.Block, error)
	GetHeaderByHash(*bc.Hash) (*types.BlockHeader, error)
	GetHeaderByHeight(uint64) (*types.BlockHeader, error)
	InMainChain(bc.Hash) bool
	ProcessBlock(*types.Block) (bool, error)
	ValidateTx(*types.Tx) (bool, error)
}

// Switch is the interface for network layer
type Switch interface {
	AddReactor(name string, reactor p2p.Reactor) p2p.Reactor
	Start() error
	Stop() error
	IsListening() bool
	DialPeerWithAddress(addr *p2p.NetAddress) error
	Peers() *p2p.PeerSet
}

// Mempool is the interface for Bytom mempool
type Mempool interface {
	GetTransactions() []*core.TxDesc
	IsDust(tx *types.Tx) bool
}

//Manager is responsible for the business layer information synchronization
type Manager struct {
	sw          Switch
	chain       Chain
	mempool     Mempool
	blockKeeper *blockKeeper
	peers       *peers.PeerSet

	txSyncCh chan *txSyncMsg
	quit     chan struct{}
	config   *cfg.Config

	eventDispatcher *event.Dispatcher
	txMsgSub        *event.Subscription
}

//NewManager create a chain sync manager.
func NewManager(config *cfg.Config, sw Switch, chain Chain, mempool Mempool, dispatcher *event.Dispatcher, peers *peers.PeerSet, fastSyncDB dbm.DB) (*Manager, error) {
	manager := &Manager{
		sw:              sw,
		mempool:         mempool,
		chain:           chain,
		blockKeeper:     newBlockKeeper(chain, peers, fastSyncDB),
		peers:           peers,
		txSyncCh:        make(chan *txSyncMsg),
		quit:            make(chan struct{}),
		config:          config,
		eventDispatcher: dispatcher,
	}

	if !config.VaultMode {
		protocolReactor := NewProtocolReactor(manager)
		manager.sw.AddReactor("PROTOCOL", protocolReactor)
	}
	return manager, nil
}

// AddPeer add the network layer peer to logic layer
func (m *Manager) AddPeer(peer peers.BasePeer) {
	m.peers.AddPeer(peer)
}

//IsCaughtUp check wheather the peer finish the sync
func (m *Manager) IsCaughtUp() bool {
	peer := m.peers.BestPeer(consensus.SFFullNode)
	return peer == nil || peer.Height() <= m.chain.BestBlockHeight()
}

func (m *Manager) handleBlockMsg(peer *peers.Peer, msg *msgs.BlockMessage) {
	block, err := msg.GetBlock()
	if err != nil {
		return
	}

	m.blockKeeper.processBlock(peer.ID(), block)
}

func (m *Manager) handleBlocksMsg(peer *peers.Peer, msg *msgs.BlocksMessage) {
	blocks, err := msg.GetBlocks()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("fail on handleBlocksMsg GetBlocks")
		return
	}

	m.blockKeeper.processBlocks(peer.ID(), blocks)
}

func (m *Manager) handleFilterAddMsg(peer *peers.Peer, msg *msgs.FilterAddMessage) {
	peer.AddFilterAddress(msg.Address)
}

func (m *Manager) handleFilterClearMsg(peer *peers.Peer) {
	peer.FilterClear()
}

func (m *Manager) handleFilterLoadMsg(peer *peers.Peer, msg *msgs.FilterLoadMessage) {
	peer.AddFilterAddresses(msg.Addresses)
}

func (m *Manager) handleGetBlockMsg(peer *peers.Peer, msg *msgs.GetBlockMessage) {
	var block *types.Block
	var err error
	if msg.Height != 0 {
		block, err = m.chain.GetBlockByHeight(msg.Height)
	} else {
		block, err = m.chain.GetBlockByHash(msg.GetHash())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetBlockMsg get block from chain")
		return
	}

	ok, err := peer.SendBlock(block)
	if !ok {
		m.peers.RemovePeer(peer.ID())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetBlockMsg sentBlock")
	}
}

func (m *Manager) handleGetBlocksMsg(peer *peers.Peer, msg *msgs.GetBlocksMessage) {
	endTime := time.Now().Add(requireBlocksTimeout / 10)
	isTimeout := func() bool {
		return time.Now().After(endTime)
	}

	blocks, err := m.blockKeeper.locateBlocks(msg.GetBlockLocator(), msg.GetStopHash(), isTimeout)
	if err != nil || len(blocks) == 0 {
		log.WithFields(log.Fields{
			"module": logModule,
			"err":    err,
			"size":   len(blocks),
		}).Error("fail on handleGetBlocksMsg locateBlocks")
		return
	}

	totalSize := 0
	sendBlocks := []*types.Block{}
	for _, block := range blocks {
		rawData, err := block.MarshalText()
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetBlocksMsg marshal block")
			return
		}

		if totalSize+len(rawData) > msgs.MaxBlockchainResponseSize/2 {
			break
		}
		totalSize += len(rawData)
		sendBlocks = append(sendBlocks, block)
	}

	ok, err := peer.SendBlocks(sendBlocks)
	if !ok {
		m.peers.RemovePeer(peer.ID())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetBlocksMsg sentBlock")
	}
}

func (m *Manager) handleGetHeadersMsg(peer *peers.Peer, msg *msgs.GetHeadersMessage) {
	headers, err := m.blockKeeper.locateHeaders(msg.GetBlockLocator(), msg.GetStopHash(), msg.GetSkip(), maxNumOfHeadersPerMsg)
	if err != nil || len(headers) == 0 {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("fail on handleGetHeadersMsg locateHeaders")
		return
	}

	ok, err := peer.SendHeaders(headers)
	if !ok {
		m.peers.RemovePeer(peer.ID())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetHeadersMsg sentBlock")
	}
}

func (m *Manager) handleGetMerkleBlockMsg(peer *peers.Peer, msg *msgs.GetMerkleBlockMessage) {
	var err error
	var block *types.Block
	if msg.Height != 0 {
		block, err = m.chain.GetBlockByHeight(msg.Height)
	} else {
		block, err = m.chain.GetBlockByHash(msg.GetHash())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetMerkleBlockMsg get block from chain")
		return
	}

	ok, err := peer.SendMerkleBlock(block)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetMerkleBlockMsg sentMerkleBlock")
		return
	}

	if !ok {
		m.peers.RemovePeer(peer.ID())
	}
}

func (m *Manager) handleHeadersMsg(peer *peers.Peer, msg *msgs.HeadersMessage) {
	headers, err := msg.GetHeaders()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("fail on handleHeadersMsg GetHeaders")
		return
	}

	m.blockKeeper.processHeaders(peer.ID(), headers)
}

func (m *Manager) handleStatusMsg(basePeer peers.BasePeer, msg *msgs.StatusMessage) {
	if peer := m.peers.GetPeer(basePeer.ID()); peer != nil {
		peer.SetBestStatus(msg.BestHeight, msg.GetBestHash())
		peer.SetJustifiedStatus(msg.JustifiedHeight, msg.GetIrreversibleHash())
	}
}

func (m *Manager) handleTransactionMsg(peer *peers.Peer, msg *msgs.TransactionMessage) {
	tx, err := msg.GetTransaction()
	if err != nil {
		m.peers.ProcessIllegal(peer.ID(), security.LevelConnException, "fail on get tx from message")
		return
	}

	if m.mempool.IsDust(tx) {
		log.WithFields(log.Fields{"tx_hash": tx.ID.String(), "peer": peer.Addr()}).Warn("receive dust tx msg")
		return
	}

	m.peers.MarkTx(peer.ID(), tx.ID)
	if isOrphan, err := m.chain.ValidateTx(tx); err != nil && err != core.ErrDustTx && !isOrphan {
		m.peers.ProcessIllegal(peer.ID(), security.LevelMsgIllegal, "fail on validate tx transaction")
	}
}

func (m *Manager) handleTransactionsMsg(peer *peers.Peer, msg *msgs.TransactionsMessage) {
	txs, err := msg.GetTransactions()
	if err != nil {
		m.peers.ProcessIllegal(peer.ID(), security.LevelConnException, "fail on get txs from message")
		return
	}

	if len(txs) > msgs.TxsMsgMaxTxNum {
		m.peers.ProcessIllegal(peer.ID(), security.LevelMsgIllegal, "exceeded the maximum tx number limit")
		return
	}

	for _, tx := range txs {
		if m.mempool.IsDust(tx) {
			m.peers.ProcessIllegal(peer.ID(), security.LevelMsgIllegal, "receive dust txs msg")
			continue
		}

		m.peers.MarkTx(peer.ID(), tx.ID)
		if isOrphan, err := m.chain.ValidateTx(tx); err != nil && !isOrphan {
			m.peers.ProcessIllegal(peer.ID(), security.LevelMsgIllegal, "fail on validate tx transaction")
			return
		}
	}
}

func (m *Manager) processMsg(basePeer peers.BasePeer, msgType byte, msg msgs.BlockchainMessage) {
	peer := m.peers.GetPeer(basePeer.ID())
	if peer == nil {
		return
	}

	log.WithFields(log.Fields{
		"module":  logModule,
		"peer":    basePeer.Addr(),
		"type":    reflect.TypeOf(msg),
		"message": msg.String(),
	}).Debug("receive message from peer")

	switch msg := msg.(type) {
	case *msgs.GetBlockMessage:
		m.handleGetBlockMsg(peer, msg)

	case *msgs.BlockMessage:
		m.handleBlockMsg(peer, msg)

	case *msgs.StatusMessage:
		m.handleStatusMsg(basePeer, msg)

	case *msgs.TransactionMessage:
		m.handleTransactionMsg(peer, msg)

	case *msgs.TransactionsMessage:
		m.handleTransactionsMsg(peer, msg)

	case *msgs.GetHeadersMessage:
		m.handleGetHeadersMsg(peer, msg)

	case *msgs.HeadersMessage:
		m.handleHeadersMsg(peer, msg)

	case *msgs.GetBlocksMessage:
		m.handleGetBlocksMsg(peer, msg)

	case *msgs.BlocksMessage:
		m.handleBlocksMsg(peer, msg)

	case *msgs.FilterLoadMessage:
		m.handleFilterLoadMsg(peer, msg)

	case *msgs.FilterAddMessage:
		m.handleFilterAddMsg(peer, msg)

	case *msgs.FilterClearMessage:
		m.handleFilterClearMsg(peer)

	case *msgs.GetMerkleBlockMessage:
		m.handleGetMerkleBlockMsg(peer, msg)

	default:
		log.WithFields(log.Fields{
			"module":       logModule,
			"peer":         basePeer.Addr(),
			"message_type": reflect.TypeOf(msg),
		}).Error("unhandled message type")
	}
}

// RemovePeer delete peer for peer set
func (m *Manager) RemovePeer(peerID string) {
	m.peers.RemovePeer(peerID)
}

// SendStatus sent the current self status to remote peer
func (m *Manager) SendStatus(peer peers.BasePeer) error {
	p := m.peers.GetPeer(peer.ID())
	if p == nil {
		return errors.New("invalid peer")
	}

	lastJustifiedHeader, err := m.chain.LastJustifiedHeader()
	if err != nil {
		return err
	}

	if err := p.SendStatus(m.chain.BestBlockHeader(), lastJustifiedHeader); err != nil {
		m.peers.RemovePeer(p.ID())
		return err
	}
	return nil
}

// Start the network logic layer
func (m *Manager) Start() error {
	var err error
	m.txMsgSub, err = m.eventDispatcher.Subscribe(core.TxMsgEvent{})
	if err != nil {
		return err
	}
	m.blockKeeper.start()
	go m.broadcastTxsLoop()
	go m.syncMempoolLoop()

	return nil
}

//Stop stop sync manager
func (m *Manager) Stop() {
	m.blockKeeper.stop()
	close(m.quit)
}
