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

func (a *API) getWorkJSON() Response {
	work, err := a.GetWorkJSON()
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(work)
}

// SubmitWorkReq used to submitWork req
type SubmitWorkReq struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
}

func (a *API) submitWork(ctx context.Context, req *SubmitWorkReq) Response {
	if err := a.SubmitWork(req.BlockHeader); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(true)
}

func (a *API) submitWorkJSON(ctx context.Context, req *SubmitWorkReq) Response {
	if err := a.SubmitWork(req.BlockHeader); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(true)
}

// GetWorkResp is resp struct for API
type GetWorkResp struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
	Seed        *bc.Hash           `json:"seed"`
}

// GetWork get work
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

type BlockHeaderJSON struct {
	Version           uint64 				`json:"version"`  // The version of the block.
	Height            uint64 				`json:"height"`  // The height of the block.
	PreviousBlockHash bc.Hash 				`json:"previous_block_hash"` // The hash of the previous block.
	Timestamp         uint64  				`json:"timestamp"` // The time of the block in seconds.
	Nonce             uint64 				`json:"nonce"` // Nonce used to generate the block.
	Bits              uint64 				`json:"bits"` // Difficulty target for the block.
	BlockCommitment   *types.BlockCommitment `json:"block_commitment"` //Block commitment
}

// GetWorkResp is resp struct for API get-work-json
type GetWorkJSONResp struct {
	BlockHeader *BlockHeaderJSON 	`json:"block_header"`
	Seed        *bc.Hash           	`json:"seed"`
}

// GetWorkJSON get work in json
func (a *API) GetWorkJSON() (*GetWorkJSONResp, error) {
	bh, err := a.miningPool.GetWork()
	if err != nil {
		return nil, err
	}

	seed, err := a.chain.CalcNextSeed(&bh.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	blockCommitment := &types.BlockCommitment{
							TransactionsMerkleRoot: bh.BlockCommitment.TransactionsMerkleRoot,
							TransactionStatusHash:  bh.BlockCommitment.TransactionStatusHash,
						}

	return &GetWorkJSONResp{
		BlockHeader: &BlockHeaderJSON{
			Version:			bh.Version,
			Height:				bh.Height,
			PreviousBlockHash:	bh.PreviousBlockHash,
			Timestamp:         	bh.Timestamp,
			Nonce:             	bh.Nonce,
			Bits:              	bh.Bits,
			BlockCommitment:	blockCommitment,
		},
		Seed:        seed,
	}, nil
}

// SubmitWork submit work
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
