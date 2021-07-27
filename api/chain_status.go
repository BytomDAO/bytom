package api

// ChainStatus indicate chain status
type ChainStatus struct {
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

	finalizedHash := finalizedBlockHeader.Hash()
	justifiedHash := justifiedBlockHeader.Hash()
	return &ChainStatus{
		FinalizedHeight: finalizedBlockHeader.Height,
		FinalizedHash:   finalizedHash.String(),
		JustifiedHeight: justifiedBlockHeader.Height,
		JustifiedHash:   justifiedHash.String(),
	}, nil
}
