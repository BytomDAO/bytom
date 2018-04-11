package api

import (
	"context"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

func (a *API) getWork() Response {
	work, err := a.GetWork()
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(work)
}

type SubmitWorkReq struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
}

func (a *API) submitWork(ctx context.Context, req *SubmitWorkReq) Response {
	if err := a.SubmitWork(req.BlockHeader); err != nil {
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

// GetWorkResp is resp struct for API
type GetWorkResp struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
	Seed        *bc.Hash           `json:"seed"`
}

func (a *API) GetWork() (*GetWorkResp, error) {
	bh, err := a.miningPool.GetWork()
	if err != nil {
		return nil, err
	}

	seed, err := a.chain.CalcNextSeed(&bh.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	return &GetWorkResp{
		BlockHeader: bh,
		Seed:        seed,
	}, nil
}

func (a *API) SubmitWork(bh *types.BlockHeader) error {
	return a.miningPool.SubmitWork(bh)
}

func (a *API) setMining(in struct {
	IsMining bool `json:"is_mining"`
}) Response {
	if in.IsMining {
		return a.startMining()
	}
	return a.stopMining()
}

func (a *API) startMining() Response {
	a.cpuMiner.Start()
	if !a.IsMining() {
		return NewErrorResponse(errors.New("Failed to start mining"))
	}
	return NewSuccessResponse("")
}

func (a *API) stopMining() Response {
	a.cpuMiner.Stop()
	if a.IsMining() {
		return NewErrorResponse(errors.New("Failed to stop mining"))
	}
	return NewSuccessResponse("")
}
