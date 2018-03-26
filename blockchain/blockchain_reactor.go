package blockchain

type NetInfo struct {
	Listening    bool   `json:"listening"`
	Syncing      bool   `json:"syncing"`
	Mining       bool   `json:"mining"`
	PeerCount    int    `json:"peer_count"`
	CurrentBlock uint64 `json:"current_block"`
	HighestBlock uint64 `json:"highest_block"`
}

func (bcr *BlockchainReactor) GetNodeInfo() *NetInfo {
	return &NetInfo{
		Listening:    bcr.sw.IsListening(),
		Syncing:      bcr.blockKeeper.IsCaughtUp(),
		Mining:       bcr.mining.IsMining(),
		PeerCount:    len(bcr.sw.Peers().List()),
		CurrentBlock: bcr.blockKeeper.chainHeight,
		HighestBlock: bcr.blockKeeper.maxPeerHeight,
	}
}

func (bcr *BlockchainReactor) IsMining() bool {
	return bcr.mining.IsMining()
}
