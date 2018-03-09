package blockchain

import (
	"context"

	"github.com/bytom/protocol/bc/legacy"
)

type BlockHeaderByHeight struct {
	BlockHeader *legacy.BlockHeader `json:"block_header"`
	Reward      uint64              `json:"reward"`
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

func (bcr *BlockchainReactor) getBlockHeaderByHeight(ctx context.Context, req struct {
	Height uint64 `json:"block_height"`
}) Response {
	block, err := bcr.chain.GetBlockByHeight(req.Height)
	if err != nil {
		return NewErrorResponse(err)
	}

	resp := &BlockHeaderByHeight{
		BlockHeader: &block.BlockHeader,
		Reward:      block.Transactions[0].Outputs[0].Amount,
	}
	return NewSuccessResponse(resp)
}
