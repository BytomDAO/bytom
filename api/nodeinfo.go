package api

import (
	"context"
	"net"

	"github.com/bytom/errors"
	"github.com/bytom/netsync"
	"github.com/bytom/p2p"
	"github.com/bytom/version"
)

// NetInfo indicate net information
type NetInfo struct {
	Listening    bool   `json:"listening"`
	Syncing      bool   `json:"syncing"`
	Mining       bool   `json:"mining"`
	PeerCount    int    `json:"peer_count"`
	CurrentBlock uint64 `json:"current_block"`
	HighestBlock uint64 `json:"highest_block"`
	NetWorkID    string `json:"network_id"`
	Version      string `json:"version"`
}

// GetNodeInfo return net information
func (a *API) GetNodeInfo() *NetInfo {
	info := &NetInfo{
		Listening:    a.sync.Switch().IsListening(),
		Syncing:      !a.sync.IsCaughtUp(),
		Mining:       a.cpuMiner.IsMining(),
		PeerCount:    len(a.sync.Switch().Peers().List()),
		CurrentBlock: a.chain.BestBlockHeight(),
		NetWorkID:    a.sync.NodeInfo().Network,
		Version:      version.Version,
	}
	if bestPeer := a.sync.BestPeer(); bestPeer != nil {
		info.HighestBlock = bestPeer.Height
	}
	if info.CurrentBlock > info.HighestBlock {
		info.HighestBlock = info.CurrentBlock
	}
	return info
}

// return the currently connected peers with net address
func (a *API) getPeerInfoByAddr(addr string) *netsync.PeerInfo {
	peerInfos := a.sync.GetPeerInfos()
	for _, peerInfo := range peerInfos {
		if peerInfo.RemoteAddr == addr {
			return peerInfo
		}
	}
	return nil
}

// disconnect peer by the peer id
func (a *API) disconnectPeerById(peerID string) error {
	return a.sync.StopPeer(peerID)
}

// connect peer b y net address
func (a *API) connectPeerByIpAndPort(ip string, port uint16) (*netsync.PeerInfo, error) {
	netIp := net.ParseIP(ip)
	if netIp == nil {
		return nil, errors.New("invalid ip address")
	}

	addr := p2p.NewNetAddressIPPort(netIp, port)
	sw := a.sync.Switch()

	if err := sw.DialPeerWithAddress(addr); err != nil {
		return nil, errors.Wrap(err, "can not connect to the address")
	}
	peer := a.getPeerInfoByAddr(addr.String())
	if peer == nil {
		return nil, errors.New("the peer is disconnected again")
	}
	return peer, nil
}

// getNetInfo return network information
func (a *API) getNetInfo() Response {
	return NewSuccessResponse(a.GetNodeInfo())
}

// isMining return is in mining or not
func (a *API) isMining() Response {
	IsMining := map[string]bool{"is_mining": a.IsMining()}
	return NewSuccessResponse(IsMining)
}

// IsMining return mining status
func (a *API) IsMining() bool {
	return a.cpuMiner.IsMining()
}

// return the peers of current node
func (a *API) listPeers() Response {
	return NewSuccessResponse(a.sync.GetPeerInfos())
}

// disconnect peer
func (a *API) disconnectPeer(ctx context.Context, ins struct {
	PeerId string `json:"peerId"`
}) Response {
	if err := a.disconnectPeerById(ins.PeerId); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(nil)
}

// connect peer by ip and port
func (a *API) connectPeer(ctx context.Context, ins struct {
	Ip   string `json:"ip"`
	Port uint16 `json:"port"`
}) Response {
	if peer, err := a.connectPeerByIpAndPort(ins.Ip, ins.Port); err != nil {
		return NewErrorResponse(err)
	} else {
		return NewSuccessResponse(peer)
	}
}
