package api

import (
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
		Syncing:      a.sync.BlockKeeper().IsCaughtUp(),
		Mining:       a.cpuMiner.IsMining(),
		PeerCount:    len(a.sync.Switch().Peers().List()),
		CurrentBlock: a.chain.BestBlockHeight(),
		NetWorkID:    a.sync.NodeInfo().Network,
		Version:      version.Version,
	}
	_, info.HighestBlock = a.sync.Peers().BestPeer()
	if info.CurrentBlock > info.HighestBlock {
		info.HighestBlock = info.CurrentBlock
	}
	return info
}

// getNetInfo return network infomation
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
