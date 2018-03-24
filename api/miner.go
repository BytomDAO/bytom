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

func (a *API) submitWork(bh *types.BlockHeader) Response {
	return NewSuccessResponse(a.bcr.SubmitWork(bh))
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
