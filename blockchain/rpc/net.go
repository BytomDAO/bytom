package rpc

import (
	ctypes "github.com/bytom/blockchain/rpc/types"
	"github.com/bytom/p2p"
)

//-----------------------------------------------------------------------------

//NetInfo return p2p net status
func NetInfo(p2pSwitch *p2p.Switch) (*ctypes.ResultNetInfo, error) {
	listening := p2pSwitch.IsListening()
	listeners := []string{}
	for _, listener := range p2pSwitch.Listeners() {
		listeners = append(listeners, listener.String())
	}
	peers := []ctypes.Peer{}
	for _, peer := range p2pSwitch.Peers().List() {
		peers = append(peers, ctypes.Peer{
			NodeInfo:         *peer.NodeInfo,
			IsOutbound:       peer.IsOutbound(),
			ConnectionStatus: peer.Connection().Status(),
		})
	}
	return &ctypes.ResultNetInfo{
		Listening: listening,
		Listeners: listeners,
		Peers:     peers,
	}, nil
}
