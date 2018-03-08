package blockchain

import (
	"context"

	chainjson "github.com/bytom/encoding/json"
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

func (bcr *BlockchainReactor) checkReward(ctx context.Context, req struct {
	HexHash chainjson.HexBytes `json:"block_hash"`
}) Response {
	var b32 [32]byte
	copy(b32[:], req.HexHash)
	hash := bc.NewHash(b32)
	reward, err := bcr.miningPool.CheckReward(&hash)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(reward)
}
