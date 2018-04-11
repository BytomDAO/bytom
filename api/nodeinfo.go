package api

type NetInfo struct {
	Listening    bool   `json:"listening"`
	Syncing      bool   `json:"syncing"`
	Mining       bool   `json:"mining"`
	PeerCount    int    `json:"peer_count"`
	CurrentBlock uint64 `json:"current_block"`
	HighestBlock uint64 `json:"highest_block"`
}

func (a *API) GetNodeInfo() *NetInfo {
	info := &NetInfo{
		Listening:    a.sync.Switch().IsListening(),
		Syncing:      a.sync.BlockKeeper().IsCaughtUp(),
		Mining:       a.cpuMiner.IsMining(),
		PeerCount:    len(a.sync.Switch().Peers().List()),
		CurrentBlock: a.chain.BestBlockHeight(),
	}
	_, info.HighestBlock = a.sync.Peers().BestPeer()
	if info.CurrentBlock > info.HighestBlock {
		info.HighestBlock = info.CurrentBlock
	}
	return info
}

// return network infomation
func (a *API) getNetInfo() Response {
	return NewSuccessResponse(a.GetNodeInfo())
}

// return is in mining or not
func (a *API) isMining() Response {
	IsMining := map[string]bool{"isMining": a.IsMining()}
	return NewSuccessResponse(IsMining)
}

func (a *API) IsMining() bool {
	return a.cpuMiner.IsMining()
}
