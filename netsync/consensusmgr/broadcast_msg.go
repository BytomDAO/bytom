package consensusmgr

import (
	"github.com/bytom/bytom/netsync/peers"
)

// BroadcastMsg the message that can be broadcast
// by peer set.
type BroadcastMsg struct {
	msg       ConsensusMessage
	transChan byte
}

// NewBroadcastMsg create concrete broadcast message, implement peers.BroadcastMsg interface.
func NewBroadcastMsg(msg ConsensusMessage, transChan byte) *BroadcastMsg {
	return &BroadcastMsg{
		msg:       msg,
		transChan: transChan,
	}
}

// GetChan get message transfer channel.
func (b *BroadcastMsg) GetChan() byte {
	return b.transChan
}

// GetMsg get ConsensusMessage.
func (b *BroadcastMsg) GetMsg() interface{} {
	return struct{ ConsensusMessage }{b.msg}
}

// MsgString get a string representation of the message.
func (b *BroadcastMsg) MsgString() string {
	return b.msg.String()
}

// MarkSendRecord mark send message record to prevent messages from being sent repeatedly.
func (b *BroadcastMsg) MarkSendRecord(ps *peers.PeerSet, peers []string) {
	b.msg.BroadcastMarkSendRecord(ps, peers)
}

// FilterTargetPeers filter target peers to filter the nodes that need to send messages.
func (b *BroadcastMsg) FilterTargetPeers(ps *peers.PeerSet) []string {
	return b.msg.BroadcastFilterTargetPeers(ps)
}
