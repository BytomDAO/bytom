package api

// ChainStatus indicate chain status
type ChainStatus struct {
	HighestHeight   uint64 `json:"highest_height"`
	CurrentHeight   uint64 `json:"current_height"`
	CurrentHash     string `json:"current_hash"`
	FinalizedHeight uint64 `json:"finalized_height"`
	FinalizedHash   string `json:"finalized_hash"`
	JustifiedHeight uint64 `json:"justified_height"`
	JustifiedHash   string `json:"justified_hash"`
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

	finalizedHash := finalizedBlockHeader.Hash()
	justifiedHash := justifiedBlockHeader.Hash()
	return &ChainStatus{
		HighestHeight:   highestBlockHeight,
		CurrentHeight:   a.chain.BestBlockHeight(),
		CurrentHash:     a.chain.BestBlockHash().String(),
		FinalizedHeight: finalizedBlockHeader.Height,
		FinalizedHash:   finalizedHash.String(),
		JustifiedHeight: justifiedBlockHeader.Height,
		JustifiedHash:   justifiedHash.String(),
	}, nil
}
