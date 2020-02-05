package p2p

import (
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/bytom/bytom/p2p/connection"
)

//Reactor is responsible for handling incoming messages of one or more `Channels`
type Reactor interface {
	cmn.Service // Start, Stop

	// SetSwitch allows setting a switch.
	SetSwitch(*Switch)

	// GetChannels returns the list of channel descriptors.
	GetChannels() []*connection.ChannelDescriptor

	// AddPeer is called by the switch when a new peer is added.
	AddPeer(peer *Peer) error

	// RemovePeer is called by the switch when the peer is stopped (due to error
	// or other reason).
	RemovePeer(peer *Peer, reason interface{})

	// Receive is called when msgBytes is received from peer.
	//
	// NOTE reactor can not keep msgBytes around after Receive completes without
	// copying.
	//
	// CONTRACT: msgBytes are not nil.
	Receive(chID byte, peer *Peer, msgBytes []byte)
}

//BaseReactor base service of a reactor
type BaseReactor struct {
	cmn.BaseService // Provides Start, Stop, .Quit
	Switch          *Switch
}

//NewBaseReactor create new base Reactor
func NewBaseReactor(name string, impl Reactor) *BaseReactor {
	return &BaseReactor{
		BaseService: *cmn.NewBaseService(nil, name, impl),
		Switch:      nil,
	}
}

//SetSwitch setting a switch for reactor
func (br *BaseReactor) SetSwitch(sw *Switch) {
	br.Switch = sw
}

//GetChannels returns the list of channel descriptors
func (*BaseReactor) GetChannels() []*connection.ChannelDescriptor { return nil }

//AddPeer is called by the switch when a new peer is added
func (*BaseReactor) AddPeer(peer *Peer) {}

//RemovePeer is called by the switch when the peer is stopped (due to error or other reason)
func (*BaseReactor) RemovePeer(peer *Peer, reason interface{}) {}

//Receive is called when msgBytes is received from peer
func (*BaseReactor) Receive(chID byte, peer *Peer, msgBytes []byte) {}
