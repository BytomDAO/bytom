package api

// ChainStatus indicate chain status
type ChainStatus struct {
	CurrentBlock   uint64 `json:"current_block"`
	HighestBlock   uint64 `json:"highest_block"`
	FinalizedBlock uint64 `json:"finalized_block"`
	JustifiedBlock uint64 `json:"justified_block"`
}

// getChainStatus return chain  status
func (a *API) getChainStatus() Response {
	chainStatus, err := a.GetChainStatus()
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(chainStatus)
}

// GetChainStatus return chain status
func (a *API) GetChainStatus() (*ChainStatus, error) {
	finalizedBlockHeader, err := a.chain.LastFinalizedHeader()
	if err != nil {
		return nil, err
	}

	justifiedBlockHeader, err := a.chain.LastJustifiedHeader()
	if err != nil {
		return nil, err
	}

	highestBlockHeight := a.chain.BestBlockHeight()
	if bestPeer := a.sync.BestPeer(); bestPeer != nil {
		if bestPeer.Height > highestBlockHeight {
			highestBlockHeight = bestPeer.Height
		}
	}

	return &ChainStatus{
		CurrentBlock:   a.chain.BestBlockHeight(),
		HighestBlock:   highestBlockHeight,
		FinalizedBlock: finalizedBlockHeader.Height,
		JustifiedBlock: justifiedBlockHeader.Height,
	}, nil
}
