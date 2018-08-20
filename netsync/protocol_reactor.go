package netsync

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
	"github.com/bytom/p2p"
	"github.com/bytom/p2p/connection"
)

const (
	handshakeTimeout    = 10 * time.Second
	handshakeCheckPerid = 500 * time.Millisecond
)

var (
	errProtocolHandshakeTimeout = errors.New("Protocol handshake timeout")
	errStatusRequest            = errors.New("Status request error")
)

//ProtocolReactor handles new coming protocol message.
type ProtocolReactor struct {
	p2p.BaseReactor

	sm    *SyncManager
	peers *peerSet
}

// NewProtocolReactor returns the reactor of whole blockchain.
func NewProtocolReactor(sm *SyncManager, peers *peerSet) *ProtocolReactor {
	pr := &ProtocolReactor{
		sm:    sm,
		peers: peers,
	}
	pr.BaseReactor = *p2p.NewBaseReactor("ProtocolReactor", pr)
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

// AddPeer implements Reactor by sending our state to peer.
func (pr *ProtocolReactor) AddPeer(peer *p2p.Peer) error {
	if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{&StatusRequestMessage{}}); !ok {
		return errStatusRequest
	}

	checkTicker := time.NewTimer(handshakeCheckPerid)
	timeoutTicker := time.NewTimer(handshakeTimeout)
	for {
		select {
		case <-checkTicker.C:
			if exist := pr.peers.getPeer(peer.Key); exist != nil {
				pr.sm.syncTransactions(peer.Key)
				return nil
			}

		case <-timeoutTicker.C:
			return errProtocolHandshakeTimeout
		}
	}
}

// RemovePeer implements Reactor by removing peer from the pool.
func (pr *ProtocolReactor) RemovePeer(peer *p2p.Peer, reason interface{}) {
	pr.peers.removePeer(peer.Key)
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (pr *ProtocolReactor) Receive(chID byte, src *p2p.Peer, msgBytes []byte) {
	msgType, msg, err := DecodeMessage(msgBytes)
	if err != nil {
		log.WithField("err", err).Errorf("fail on reactor decoding message")
		return
	}

	pr.sm.processMsg(src, msgType, msg)
}
