package api

import (
	"context"
	"net"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/p2p"
	"github.com/bytom/bytom/version"
)

type VersionInfo struct {
	Version string `json:"version"`
	Update  uint16 `json:"update"` // 0: no update; 1: small update; 2: significant update
	NewVer  string `json:"new_version"`
}

// NetInfo indicate net information
type NetInfo struct {
	Listening     bool         `json:"listening"`
	Syncing       bool         `json:"syncing"`
	Mining        bool         `json:"mining"`
	NodeXPub      string       `json:"node_xpub"`
	PeerCount     int          `json:"peer_count"`
	HighestHeight uint64       `json:"highest_height"`
	NetWorkID     string       `json:"network_id"`
	Version       *VersionInfo `json:"version_info"`
}

// getNetInfo return network information
func (a *API) getNetInfo() Response {
	highestBlockHeight := a.chain.BestBlockHeight()
	if bestPeer := a.sync.BestPeer(); bestPeer != nil {
		if bestPeer.Height > highestBlockHeight {
			highestBlockHeight = bestPeer.Height
		}
	}
	return NewSuccessResponse(&NetInfo{
		Listening:     a.sync.IsListening(),
		Syncing:       !a.sync.IsCaughtUp(),
		Mining:        a.blockProposer.IsProposing(),
		NodeXPub:      config.CommonConfig.PrivateKey().XPub().String(),
		PeerCount:     a.sync.PeerCount(),
		HighestHeight: highestBlockHeight,
		NetWorkID:     a.sync.GetNetwork(),
		Version: &VersionInfo{
			Version: version.Version,
			Update:  version.Status.VersionStatus(),
			NewVer:  version.Status.MaxVerSeen(),
		},
	})
}

// return the currently connected peers with net address
func (a *API) getPeerInfoByAddr(addr string) *peers.PeerInfo {
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
func (a *API) connectPeerByIpAndPort(ip string, port uint16) (*peers.PeerInfo, error) {
	netIp := net.ParseIP(ip)
	if netIp == nil {
		return nil, errors.New("invalid ip address")
	}

	addr := p2p.NewNetAddressIPPort(netIp, port)

	if err := a.sync.DialPeerWithAddress(addr); err != nil {
		return nil, errors.Wrap(err, "can not connect to the address")
	}
	peer := a.getPeerInfoByAddr(addr.String())
	if peer == nil {
		return nil, errors.New("the peer is disconnected again")
	}
	return peer, nil
}

// isMining return is in mining or not
func (a *API) isMining() Response {
	IsMining := map[string]bool{"is_mining": a.blockProposer.IsProposing()}
	return NewSuccessResponse(IsMining)
}

// IsMining return mining status
func (a *API) IsMining() bool {
	return a.blockProposer.IsProposing()
}

// return the peers of current node
func (a *API) listPeers() Response {
	return NewSuccessResponse(a.sync.GetPeerInfos())
}

// disconnect peer
func (a *API) disconnectPeer(ctx context.Context, ins struct {
	PeerID string `json:"peer_id"`
}) Response {
	if err := a.disconnectPeerById(ins.PeerID); err != nil {
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
