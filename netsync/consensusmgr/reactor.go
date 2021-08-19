package consensusmgr

import (
	"github.com/sirupsen/logrus"

	"github.com/bytom/bytom/p2p"
	"github.com/bytom/bytom/p2p/connection"
)

const (
	logModule                 = "consensus"
	consensusChannel          = byte(0x50)
	maxBlockchainResponseSize = 22020096 + 2
)

// ConsensusReactor handles new coming consensus message.
type ConsensusReactor struct {
	p2p.BaseReactor
	manager *Manager
}

// NewConsensusReactor create consensus reactor.
func NewConsensusReactor(manager *Manager) *ConsensusReactor {
	cr := &ConsensusReactor{
		manager: manager,
	}
	cr.BaseReactor = *p2p.NewBaseReactor("ConsensusReactor", cr)
	return cr
}

// GetChannels implements Reactor
func (cr *ConsensusReactor) GetChannels() []*connection.ChannelDescriptor {
	return []*connection.ChannelDescriptor{
		{
			ID:                consensusChannel,
			Priority:          10,
			SendQueueCapacity: 100,
		},
	}
}

// OnStart implements BaseService
func (cr *ConsensusReactor) OnStart() error {
	return cr.BaseReactor.OnStart()
}

// OnStop implements BaseService
func (cr *ConsensusReactor) OnStop() {
	cr.BaseReactor.OnStop()
}

// AddPeer implements Reactor by sending our state to peer.
func (cr *ConsensusReactor) AddPeer(peer *p2p.Peer) error {
	cr.manager.addPeer(peer)
	return nil
}

// RemovePeer implements Reactor by removing peer from the pool.
func (cr *ConsensusReactor) RemovePeer(peer *p2p.Peer, reason interface{}) {
	cr.manager.removePeer(peer.Key)
}

// Receive implements Reactor by handling messages.
func (cr *ConsensusReactor) Receive(chID byte, src *p2p.Peer, msgBytes []byte) {
	msgType, msg, err := decodeMessage(msgBytes)
	if err != nil {
		logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Error("fail on reactor decoding message")
		return
	}

	cr.manager.processMsg(src.ID(), msgType, msg)
}
