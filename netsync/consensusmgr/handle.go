package consensusmgr

import (
	"encoding/hex"
	"reflect"

	"github.com/sirupsen/logrus"

	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/p2p"
	"github.com/bytom/bytom/p2p/security"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/casper"
)

// Switch is the interface for p2p switch.
type Switch interface {
	AddReactor(name string, reactor p2p.Reactor) p2p.Reactor
}

// Chain is the interface for Bytom core.
type Chain interface {
	BestBlockHeight() uint64
	GetHeaderByHash(*bc.Hash) (*types.BlockHeader, error)
	ProcessBlock(*types.Block) (bool, error)
	ProcessBlockVerification(*casper.ValidCasperSignMsg) error
}

type Peers interface {
	AddPeer(peer peers.BasePeer)
	BroadcastMsg(bm peers.BroadcastMsg) error
	GetPeer(id string) *peers.Peer
	MarkBlock(peerID string, hash *bc.Hash)
	MarkBlockVerification(peerID string, signature []byte)
	ProcessIllegal(peerID string, level byte, reason string)
	RemovePeer(peerID string)
	SetStatus(peerID string, height uint64, hash *bc.Hash)
}

type blockMsg struct {
	block  *types.Block
	peerID string
}

// Manager is the consensus message network synchronization manager.
type Manager struct {
	sw              Switch
	chain           Chain
	peers           Peers
	blockFetcher    *blockFetcher
	eventDispatcher *event.Dispatcher

	quit chan struct{}
}

// NewManager create new manager.
func NewManager(sw Switch, chain Chain, peers Peers, dispatcher *event.Dispatcher) *Manager {
	manager := &Manager{
		sw:              sw,
		chain:           chain,
		peers:           peers,
		blockFetcher:    newBlockFetcher(chain, peers),
		eventDispatcher: dispatcher,
		quit:            make(chan struct{}),
	}
	protocolReactor := NewConsensusReactor(manager)
	manager.sw.AddReactor("CONSENSUS", protocolReactor)
	return manager
}

func (m *Manager) addPeer(peer peers.BasePeer) {
	m.peers.AddPeer(peer)
}

func (m *Manager) processMsg(peerID string, msgType byte, msg ConsensusMessage) {
	peer := m.peers.GetPeer(peerID)
	if peer == nil {
		return
	}

	logrus.WithFields(logrus.Fields{"module": logModule, "peer": peer.Addr(), "type": reflect.TypeOf(msg), "message": msg.String()}).Debug("receive message from peer")

	switch msg := msg.(type) {
	case *BlockProposeMsg:
		m.handleBlockProposeMsg(peerID, msg)

	case *BlockVerificationMsg:
		m.handleBlockVerificationMsg(peerID, msg)

	default:
		logrus.WithFields(logrus.Fields{"module": logModule, "peer": peerID, "message_type": reflect.TypeOf(msg)}).Error("unhandled message type")
	}
}

func (m *Manager) handleBlockProposeMsg(peerID string, msg *BlockProposeMsg) {
	block, err := msg.GetProposeBlock()
	if err != nil {
		logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Warning("failed on get propose block")
		return
	}

	hash := block.Hash()
	m.peers.MarkBlock(peerID, &hash)
	m.blockFetcher.processNewBlock(&blockMsg{peerID: peerID, block: block})
	m.peers.SetStatus(peerID, block.Height, &hash)
}

func (m *Manager) handleBlockVerificationMsg(peerID string, msg *BlockVerificationMsg) {
	m.peers.MarkBlockVerification(peerID, msg.Signature)
	if err := m.chain.ProcessBlockVerification(&casper.ValidCasperSignMsg{
		SourceHash: msg.SourceHash,
		TargetHash: msg.TargetHash,
		Signature:  msg.Signature,
		PubKey:     hex.EncodeToString(msg.PubKey),
	}); err != nil {
		m.peers.ProcessIllegal(peerID, security.LevelMsgIllegal, err.Error())
	}
}

func (m *Manager) blockProposeMsgBroadcastLoop() {
	m.msgBroadcastLoop(event.NewProposedBlockEvent{}, func(data interface{}) (ConsensusMessage, error) {
		ev := data.(event.NewProposedBlockEvent)
		return NewBlockProposeMsg(&ev.Block)
	})
}

func (m *Manager) blockVerificationMsgBroadcastLoop() {
	m.msgBroadcastLoop(casper.ValidCasperSignMsg{}, func(data interface{}) (ConsensusMessage, error) {
		v := data.(casper.ValidCasperSignMsg)
		pubKey, err := hex.DecodeString(v.PubKey)
		if err != nil {
			return nil, err
		}

		return NewBlockVerificationMsg(v.SourceHash, v.TargetHash, pubKey, v.Signature), nil
	})
}

func (m *Manager) msgBroadcastLoop(msgType interface{}, newMsg func(event interface{}) (ConsensusMessage, error)) {
	subscribeType := reflect.TypeOf(msgType)
	msgSub, err := m.eventDispatcher.Subscribe(msgType)
	if err != nil {
		logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Errorf("failed on subscribe %s", subscribeType)
		return
	}
	defer msgSub.Unsubscribe()
	for {
		select {
		case obj, ok := <-msgSub.Chan():
			if !ok {
				logrus.WithFields(logrus.Fields{"module": logModule}).Warningf("%sSub channel closed", subscribeType)
				return
			}

			if reflect.TypeOf(obj.Data) != subscribeType {
				logrus.WithFields(logrus.Fields{"module": logModule}).Error("event type error")
				continue
			}

			msg, err := newMsg(obj.Data)
			if err != nil {
				logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Errorf("failed on create %s message", subscribeType)
				return
			}

			message := NewBroadcastMsg(msg, consensusChannel)
			if err := m.peers.BroadcastMsg(message); err != nil {
				logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Errorf("failed on broadcast %s message.", subscribeType)
				continue
			}

		case <-m.quit:
			return
		}
	}
}

func (m *Manager) removePeer(peerID string) {
	m.peers.RemovePeer(peerID)
}

// Start consensus manager service.
func (m *Manager) Start() error {
	go m.blockFetcher.blockProcessorLoop()
	go m.blockProposeMsgBroadcastLoop()
	go m.blockVerificationMsgBroadcastLoop()
	return nil
}

// Stop consensus manager service.
func (m *Manager) Stop() {
	close(m.quit)
}
