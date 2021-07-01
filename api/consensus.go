package api

import (
	"github.com/bytom/bytom/encoding/json"
)

type VoteInfo struct {
	Vote    string `json:"vote"`
	VoteNum uint64 `json:"vote_number"`
}

func (a *API) getVoteResult(req struct {
	BlockHash   json.HexBytes `json:"block_hash"`
	BlockHeight uint64             `json:"block_height"`
}) Response {
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

