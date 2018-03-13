package blockchain

import (
	"context"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

// BlockHeaderByHeight is resp struct for API
type BlockHeaderByHeight struct {
	BlockHeader *legacy.BlockHeader `json:"block_header"`
	Reward      uint64              `json:"reward"`
}

// GetWorkResp is resp struct for API
type GetWorkResp struct {
	BlockHeader *legacy.BlockHeader `json:"block_header"`
	Seed        *bc.Hash            `json:"seed"`
}

func (bcr *BlockchainReactor) getWork() Response {
	bh, err := bcr.miningPool.GetWork()
	if err != nil {
		return NewErrorResponse(err)
	}

	seed, err := bcr.chain.GetSeed(bh.Height, &bh.PreviousBlockHash)
	if err != nil {
		return NewErrorResponse(err)
	}

	resp := &GetWorkResp{
		BlockHeader: bh,
		Seed:        seed,
	}
	return NewSuccessResponse(resp)
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
