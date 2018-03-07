package blockchain

import (
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

type WorkResp struct {
	Header legacy.BlockHeader `json:"header"`
}

func (bcr *BlockchainReactor) getWork() Response {
	bh, err := bcr.miningPool.GetWork()
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(bh)
}

func (bcr *BlockchainReactor) submitWork(bh *legacy.BlockHeader) Response {
	success := bcr.miningPool.SubmitWork(bh)
	return NewSuccessResponse(success)
}

func (bcr *BlockchainReactor) checkReward(hash *bc.Hash) Response {
	reward, err := bcr.miningPool.CheckReward(hash)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(reward)
}
