package netsync

import (
	"errors"
	"reflect"

	log "github.com/sirupsen/logrus"

	cfg "github.com/bytom/config"
	"github.com/bytom/consensus"
	"github.com/bytom/event"
	"github.com/bytom/p2p"
	core "github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	logModule             = "netsync"
	maxTxChanSize         = 10000
	maxFilterAddressSize  = 50
	maxFilterAddressCount = 1000
)

var (
	errInvalidSeedIP   = errors.New("seed ip is invalid")
	errInvalidSeedPort = errors.New("seed port is invalid")
)

// Chain is the interface for Bytom core
type Chain interface {
	BestBlockHeader() *types.BlockHeader
	BestBlockHeight() uint64
	CalcNextSeed(*bc.Hash) (*bc.Hash, error)
	GetBlockByHash(*bc.Hash) (*types.Block, error)
	GetBlockByHeight(uint64) (*types.Block, error)
	GetHeaderByHash(*bc.Hash) (*types.BlockHeader, error)
	GetHeaderByHeight(uint64) (*types.BlockHeader, error)
	GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error)
	InMainChain(bc.Hash) bool
	ProcessBlock(*types.Block) (bool, error)
	ValidateTx(*types.Tx) (bool, error)
}

type Switch interface {
	AddReactor(name string, reactor p2p.Reactor) p2p.Reactor
	AddBannedPeer(string) error
	StopPeerGracefully(string)
	NodeInfo() *p2p.NodeInfo
	Start() (bool, error)
	Stop() bool
}

//SyncManager Sync Manager is responsible for the business layer information synchronization
type SyncManager struct {
	sw           Switch
	genesisHash  bc.Hash
	chain        Chain
	txPool       *core.TxPool
	blockFetcher *blockFetcher
	blockKeeper  *blockKeeper
	peers        *peerSet

	newTxCh  chan *types.Tx
	txSyncCh chan *txSyncMsg
	quitSync chan struct{}
	config   *cfg.Config

	eventDispatcher *event.Dispatcher
	minedBlockSub   *event.Subscription
}

//NewSyncManager create a sync manager
func NewSyncManager(config *cfg.Config, sw Switch, chain Chain, txPool *core.TxPool, dispatcher *event.Dispatcher) (*SyncManager, error) {
	genesisHeader, err := chain.GetHeaderByHeight(0)
	if err != nil {
		return nil, err
	}
	peers := newPeerSet(sw)
	manager := &SyncManager{
		sw:              sw,
		genesisHash:     genesisHeader.Hash(),
		txPool:          txPool,
		chain:           chain,
		blockFetcher:    newBlockFetcher(chain, peers),
		blockKeeper:     newBlockKeeper(chain, peers),
		peers:           peers,
		newTxCh:         make(chan *types.Tx, maxTxChanSize),
		txSyncCh:        make(chan *txSyncMsg),
		quitSync:        make(chan struct{}),
		config:          config,
		eventDispatcher: dispatcher,
	}
	protocolReactor := NewProtocolReactor(manager, peers)
	manager.sw.AddReactor("PROTOCOL", protocolReactor)
	return manager, nil
}

//BestPeer return the highest p2p peerInfo
func (sm *SyncManager) BestPeer() *PeerInfo {
	bestPeer := sm.peers.bestPeer(consensus.SFFullNode)
	if bestPeer != nil {
		return bestPeer.getPeerInfo()
	}
	return nil
}

// GetNewTxCh return a unconfirmed transaction feed channel
func (sm *SyncManager) GetNewTxCh() chan *types.Tx {
	return sm.newTxCh
}

//GetPeerInfos return peer info of all peers
func (sm *SyncManager) GetPeerInfos() []*PeerInfo {
	return sm.peers.getPeerInfos()
}

//IsCaughtUp check wheather the peer finish the sync
func (sm *SyncManager) IsCaughtUp() bool {
	peer := sm.peers.bestPeer(consensus.SFFullNode)
	return peer == nil || peer.Height() <= sm.chain.BestBlockHeight()
}

//StopPeer try to stop peer by given ID
func (sm *SyncManager) StopPeer(peerID string) error {
	if peer := sm.peers.getPeer(peerID); peer == nil {
		return errors.New("peerId not exist")
	}
	sm.peers.removePeer(peerID)
	return nil
}

//Switch get sync manager switch
func (sm *SyncManager) Switch() *p2p.Switch {
	return sm.sw.(*p2p.Switch)
}

func (sm *SyncManager) handleBlockMsg(peer *peer, msg *BlockMessage) {
	block, err := msg.GetBlock()
	if err != nil {
		return
	}
	sm.blockKeeper.processBlock(peer.ID(), block)
}

func (sm *SyncManager) handleBlocksMsg(peer *peer, msg *BlocksMessage) {
	blocks, err := msg.GetBlocks()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("fail on handleBlocksMsg GetBlocks")
		return
	}

	sm.blockKeeper.processBlocks(peer.ID(), blocks)
}

func (sm *SyncManager) handleFilterAddMsg(peer *peer, msg *FilterAddMessage) {
	peer.addFilterAddress(msg.Address)
}

func (sm *SyncManager) handleFilterClearMsg(peer *peer) {
	peer.filterAdds.Clear()
}

func (sm *SyncManager) handleFilterLoadMsg(peer *peer, msg *FilterLoadMessage) {
	peer.addFilterAddresses(msg.Addresses)
}

func (sm *SyncManager) handleGetBlockMsg(peer *peer, msg *GetBlockMessage) {
	var block *types.Block
	var err error
	if msg.Height != 0 {
		block, err = sm.chain.GetBlockByHeight(msg.Height)
	} else {
		block, err = sm.chain.GetBlockByHash(msg.GetHash())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetBlockMsg get block from chain")
		return
	}

	ok, err := peer.sendBlock(block)
	if !ok {
		sm.peers.removePeer(peer.ID())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetBlockMsg sentBlock")
	}
}

func (sm *SyncManager) handleGetBlocksMsg(peer *peer, msg *GetBlocksMessage) {
	blocks, err := sm.blockKeeper.locateBlocks(msg.GetBlockLocator(), msg.GetStopHash())
	if err != nil || len(blocks) == 0 {
		return
	}

	totalSize := 0
	sendBlocks := []*types.Block{}
	for _, block := range blocks {
		rawData, err := block.MarshalText()
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetBlocksMsg marshal block")
			continue
		}

		if totalSize+len(rawData) > maxBlockchainResponseSize/2 {
			break
		}
		totalSize += len(rawData)
		sendBlocks = append(sendBlocks, block)
	}

	ok, err := peer.sendBlocks(sendBlocks)
	if !ok {
		sm.peers.removePeer(peer.ID())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetBlocksMsg sentBlock")
	}
}

func (sm *SyncManager) handleGetHeadersMsg(peer *peer, msg *GetHeadersMessage) {
	headers, err := sm.blockKeeper.locateHeaders(msg.GetBlockLocator(), msg.GetStopHash())
	if err != nil || len(headers) == 0 {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("fail on handleGetHeadersMsg locateHeaders")
		return
	}

	ok, err := peer.sendHeaders(headers)
	if !ok {
		sm.peers.removePeer(peer.ID())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetHeadersMsg sentBlock")
	}
}

func (sm *SyncManager) handleGetMerkleBlockMsg(peer *peer, msg *GetMerkleBlockMessage) {
	var err error
	var block *types.Block
	if msg.Height != 0 {
		block, err = sm.chain.GetBlockByHeight(msg.Height)
	} else {
		block, err = sm.chain.GetBlockByHash(msg.GetHash())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetMerkleBlockMsg get block from chain")
		return
	}

	blockHash := block.Hash()
	txStatus, err := sm.chain.GetTransactionStatus(&blockHash)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetMerkleBlockMsg get transaction status")
		return
	}

	ok, err := peer.sendMerkleBlock(block, txStatus)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetMerkleBlockMsg sentMerkleBlock")
		return
	}

	if !ok {
		sm.peers.removePeer(peer.ID())
	}
}

func (sm *SyncManager) handleHeadersMsg(peer *peer, msg *HeadersMessage) {
	headers, err := msg.GetHeaders()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("fail on handleHeadersMsg GetHeaders")
		return
	}

	sm.blockKeeper.processHeaders(peer.ID(), headers)
}

func (sm *SyncManager) handleMineBlockMsg(peer *peer, msg *MineBlockMessage) {
	block, err := msg.GetMineBlock()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleMineBlockMsg GetMineBlock")
		return
	}

	hash := block.Hash()
	peer.markBlock(&hash)
	sm.blockFetcher.processNewBlock(&blockMsg{peerID: peer.ID(), block: block})
	peer.setStatus(block.Height, &hash)
}

func (sm *SyncManager) handleStatusRequestMsg(peer BasePeer) {
	bestHeader := sm.chain.BestBlockHeader()
	genesisBlock, err := sm.chain.GetBlockByHeight(0)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleStatusRequestMsg get genesis")
	}

	genesisHash := genesisBlock.Hash()
	msg := NewStatusResponseMessage(bestHeader, &genesisHash)
	if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
		sm.peers.removePeer(peer.ID())
	}
}

func (sm *SyncManager) handleStatusResponseMsg(basePeer BasePeer, msg *StatusResponseMessage) {
	if peer := sm.peers.getPeer(basePeer.ID()); peer != nil {
		peer.setStatus(msg.Height, msg.GetHash())
		return
	}

	if genesisHash := msg.GetGenesisHash(); sm.genesisHash != *genesisHash {
		log.WithFields(log.Fields{
			"module":         logModule,
			"remote genesis": genesisHash.String(),
			"local genesis":  sm.genesisHash.String(),
		}).Warn("fail hand shake due to differnt genesis")
		return
	}

	sm.peers.addPeer(basePeer, msg.Height, msg.GetHash())
}

func (sm *SyncManager) handleTransactionMsg(peer *peer, msg *TransactionMessage) {
	tx, err := msg.GetTransaction()
	if err != nil {
		sm.peers.addBanScore(peer.ID(), 0, 10, "fail on get tx from message")
		return
	}

	if isOrphan, err := sm.chain.ValidateTx(tx); err != nil && !isOrphan {
		sm.peers.addBanScore(peer.ID(), 10, 0, "fail on validate tx transaction")
	}
}

func (sm *SyncManager) processMsg(basePeer BasePeer, msgType byte, msg BlockchainMessage) {
	peer := sm.peers.getPeer(basePeer.ID())
	if peer == nil && msgType != StatusResponseByte && msgType != StatusRequestByte {
		return
	}

	log.WithFields(log.Fields{
		"module":  logModule,
		"peer":    basePeer.Addr(),
		"type":    reflect.TypeOf(msg),
		"message": msg.String(),
	}).Info("receive message from peer")

	switch msg := msg.(type) {
	case *GetBlockMessage:
		sm.handleGetBlockMsg(peer, msg)

	case *BlockMessage:
		sm.handleBlockMsg(peer, msg)

	case *StatusRequestMessage:
		sm.handleStatusRequestMsg(basePeer)

	case *StatusResponseMessage:
		sm.handleStatusResponseMsg(basePeer, msg)

	case *TransactionMessage:
		sm.handleTransactionMsg(peer, msg)

	case *MineBlockMessage:
		sm.handleMineBlockMsg(peer, msg)

	case *GetHeadersMessage:
		sm.handleGetHeadersMsg(peer, msg)

	case *HeadersMessage:
		sm.handleHeadersMsg(peer, msg)

	case *GetBlocksMessage:
		sm.handleGetBlocksMsg(peer, msg)

	case *BlocksMessage:
		sm.handleBlocksMsg(peer, msg)

	case *FilterLoadMessage:
		sm.handleFilterLoadMsg(peer, msg)

	case *FilterAddMessage:
		sm.handleFilterAddMsg(peer, msg)

	case *FilterClearMessage:
		sm.handleFilterClearMsg(peer)

	case *GetMerkleBlockMessage:
		sm.handleGetMerkleBlockMsg(peer, msg)

	default:
		log.WithFields(log.Fields{
			"module":       logModule,
			"peer":         basePeer.Addr(),
			"message_type": reflect.TypeOf(msg),
		}).Error("unhandled message type")
	}
}

func (sm *SyncManager) Start() error {
	var err error
	if _, err = sm.sw.Start(); err != nil {
		log.Error("switch start err")
		return err
	}

	// broadcast transactions
	go sm.txBroadcastLoop()

	sm.minedBlockSub, err = sm.eventDispatcher.Subscribe(event.NewMinedBlockEvent{})
	if err != nil {
		return err
	}

	go sm.minedBroadcastLoop()
	go sm.txSyncLoop()

	return nil
}

//Stop stop sync manager
func (sm *SyncManager) Stop() {
	close(sm.quitSync)
	sm.minedBlockSub.Unsubscribe()
	sm.sw.Stop()
}

func (sm *SyncManager) minedBroadcastLoop() {
	for {
		select {
		case obj, ok := <-sm.minedBlockSub.Chan():
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Warning("mined block subscription channel closed")
				return
			}

			ev, ok := obj.Data.(event.NewMinedBlockEvent)
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Error("event type error")
				continue
			}

			if err := sm.peers.broadcastMinedBlock(ev.Block); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on broadcast mine block")
				continue
			}

		case <-sm.quitSync:
			return
		}
	}
}
