package blockchain

import (
	"context"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// BlockHeaderByHeight is resp struct for API
type BlockHeaderByHeight struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
	Reward      uint64             `json:"reward"`
}

// GetWorkResp is resp struct for API
type GetWorkResp struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
	Seed        *bc.Hash           `json:"seed"`
}

func (a *API) getWork() Response {
	bh, err := a.bcr.miningPool.GetWork()
	if err != nil {
		return NewErrorResponse(err)
	}

	seed, err := a.bcr.chain.GetSeed(bh.Height, &bh.PreviousBlockHash)
	if err != nil {
		return NewErrorResponse(err)
	}

	resp := &GetWorkResp{
		BlockHeader: bh,
		Seed:        seed,
	}
	return NewSuccessResponse(resp)
}

func (a *API) submitWork(bh *types.BlockHeader) Response {
	success := a.bcr.miningPool.SubmitWork(bh)
	return NewSuccessResponse(success)
}

func (a *API) getBlockHeaderByHeight(ctx context.Context, req struct {
	Height uint64 `json:"block_height"`
}) Response {
	block, err := a.bcr.chain.GetBlockByHeight(req.Height)
	if err != nil {
		return NewErrorResponse(err)
	}

	resp := &BlockHeaderByHeight{
		BlockHeader: &block.BlockHeader,
		Reward:      block.Transactions[0].Outputs[0].Amount,
	}
	return NewSuccessResponse(resp)
}
