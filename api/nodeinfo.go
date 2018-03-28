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
	return &NetInfo{
		Listening:    a.sync.Switch().IsListening(),
		Syncing:      a.sync.BlockKeeper().IsCaughtUp(),
		Mining:       a.cpuMiner.IsMining(),
		PeerCount:    len(a.sync.Switch().Peers().List()),
		CurrentBlock: a.sync.BlockKeeper().GetChainHeight(),
		HighestBlock: a.sync.BlockKeeper().BestHeight(),
	}
}

func (a *API) IsMining() bool {
	return a.cpuMiner.IsMining()
}

