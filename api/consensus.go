package api

type VoteInfo struct {
	Vote    string `json:"vote"`
	VoteNum uint64 `json:"vote_number"`
}

func (a *API) getVoteResult(req BlockReq) Response {
	blockHash := hexBytesToHash(req.BlockHash)
	if len(req.BlockHash) != 32 {
		blockHeader, err := a.chain.GetHeaderByHeight(req.BlockHeight)
		if err != nil {
			return NewErrorResponse(err)
		}

		blockHash = blockHeader.Hash()
	}

	validators, err := a.chain.AllValidators(&blockHash)
	if err != nil {
		return NewErrorResponse(err)
	}

	voteInfos := []*VoteInfo{}
	for _, validator := range validators {
		voteInfos = append(voteInfos, &VoteInfo{
			Vote:    validator.PubKey,
			VoteNum: validator.VoteNum,
		})
	}
	return NewSuccessResponse(voteInfos)
}

type ValidatorInfo struct {
	PubKey string `json:"pub_key"`
}

func (a *API) getValidatorResult(req BlockReq) Response {
	block, err := a.getBlockHelper(req)
	if err != nil {
		return NewErrorResponse(err)
	}

	var validatorPubKey string
	if block.Height > 0 {
		validator, err := a.chain.GetValidator(&block.PreviousBlockHash, block.Timestamp)
		if err != nil {
			return NewErrorResponse(err)
		}

		validatorPubKey = validator.PubKey
	}

	return NewSuccessResponse(&ValidatorInfo{PubKey: validatorPubKey})
}
