package blockchain

import (
	"context"

	"github.com/bytom/api"
	"github.com/bytom/protocol/bc/types"
)

func (bcr *BlockchainReactor) GetWork() (*api.GetWorkResp, error) {
	bh, err := bcr.miningPool.GetWork()
	if err != nil {
		return nil, err
	}

	seed, err := bcr.chain.GetSeed(bh.Height, &bh.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	return &api.GetWorkResp{
		BlockHeader: bh,
		Seed:        seed,
	}, nil
}

func (bcr *BlockchainReactor) SubmitWork(bh *types.BlockHeader) bool {
	return bcr.miningPool.SubmitWork(bh)
}

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

	resp := &api.BlockHeaderByHeight{
		BlockHeader: &block.BlockHeader,
		Reward:      block.Transactions[0].Outputs[0].Amount,
	}
	return NewSuccessResponse(resp)
}
