package api

import (
	"context"

	"github.com/bytom/protocol/bc/types"
)

func (a *API) getWork() Response {
	work, err := a.bcr.GetWork()
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(work)
}

type SubmitWorkReq struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
}

func (a *API) submitWork(ctx context.Context, req *SubmitWorkReq) Response {
	if err := a.bcr.SubmitWork(req.BlockHeader); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(true)
}

func (a *API) getBlockHeaderByHeight(ctx context.Context, req struct {
	Height uint64 `json:"block_height"`
}) Response {
	block, err := a.chain.GetBlockByHeight(req.Height)
	if err != nil {
		return NewErrorResponse(err)
	}

	resp := &BlockHeaderByHeight{
		BlockHeader: &block.BlockHeader,
		Reward:      block.Transactions[0].Outputs[0].Amount,
	}
	return NewSuccessResponse(resp)
}
