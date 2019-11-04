package api

import (
	"context"
	"strconv"

	chainjson "github.com/bytom/bytom/encoding/json"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

// BlockHeaderJSON struct provides support for get work in json format, when it also follows
// BlockHeader structure
type BlockHeaderJSON struct {
	Version           uint64                 `json:"version"`             // The version of the block.
	Height            uint64                 `json:"height"`              // The height of the block.
	PreviousBlockHash bc.Hash                `json:"previous_block_hash"` // The hash of the previous block.
	Timestamp         uint64                 `json:"timestamp"`           // The time of the block in seconds.
	Nonce             uint64                 `json:"nonce"`               // Nonce used to generate the block.
	Bits              uint64                 `json:"bits"`                // Difficulty target for the block.
	BlockCommitment   *types.BlockCommitment `json:"block_commitment"`    // Block commitment
}

type CoinbaseArbitrary struct {
	Arbitrary chainjson.HexBytes `json:"arbitrary"`
}

func (a *API) getCoinbaseArbitrary() Response {
	arbitrary := a.wallet.AccountMgr.GetCoinbaseArbitrary()
	resp := &CoinbaseArbitrary{
		Arbitrary: arbitrary,
	}
	return NewSuccessResponse(resp)
}

// setCoinbaseArbitrary add arbitary data to the reserved coinbase data.
// check function createCoinbaseTx in mining/mining.go for detail.
// arbitraryLenLimit is 107 and can be calculated by:
// 	maxHeight := ^uint64(0)
// 	reserved := append([]byte{0x00}, []byte(strconv.FormatUint(maxHeight, 10))...)
// 	arbitraryLenLimit := consensus.CoinbaseArbitrarySizeLimit - len(reserved)
func (a *API) setCoinbaseArbitrary(ctx context.Context, req CoinbaseArbitrary) Response {
	arbitraryLenLimit := 107
	if len(req.Arbitrary) > arbitraryLenLimit {
		err := errors.New("Arbitrary exceeds limit: " + strconv.FormatUint(uint64(arbitraryLenLimit), 10))
		return NewErrorResponse(err)
	}
	a.wallet.AccountMgr.SetCoinbaseArbitrary(req.Arbitrary)
	return a.getCoinbaseArbitrary()
}

// getWork gets work in compressed protobuf format
func (a *API) getWork() Response {
	work, err := a.GetWork()
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(work)
}

// getWorkJSON gets work in json format
func (a *API) getWorkJSON() Response {
	work, err := a.GetWorkJSON()
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(work)
}

// SubmitBlockReq is req struct for submit-block API
type SubmitBlockReq struct {
	Block *types.Block `json:"raw_block"`
}

// submitBlock trys to submit a raw block to the chain
func (a *API) submitBlock(ctx context.Context, req *SubmitBlockReq) Response {
	isOrphan, err := a.chain.ProcessBlock(req.Block)
	if err != nil {
		return NewErrorResponse(err)
	}

	if isOrphan {
		return NewErrorResponse(errors.New("block submitted is orphan"))
	}

	if err = a.eventDispatcher.Post(event.NewMinedBlockEvent{Block: *req.Block}); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(true)
}

// SubmitWorkReq is req struct for submit-work API
type SubmitWorkReq struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
}

// submitWork submits work in compressed protobuf format
func (a *API) submitWork(ctx context.Context, req *SubmitWorkReq) Response {
	if err := a.SubmitWork(req.BlockHeader); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(true)
}

// SubmitWorkJSONReq is req struct for submit-work-json API
type SubmitWorkJSONReq struct {
	BlockHeader *BlockHeaderJSON `json:"block_header"`
}

// submitWorkJSON submits work in json format
func (a *API) submitWorkJSON(ctx context.Context, req *SubmitWorkJSONReq) Response {
	bh := &types.BlockHeader{
		Version:           req.BlockHeader.Version,
		Height:            req.BlockHeader.Height,
		PreviousBlockHash: req.BlockHeader.PreviousBlockHash,
		Timestamp:         req.BlockHeader.Timestamp,
		Nonce:             req.BlockHeader.Nonce,
		Bits:              req.BlockHeader.Bits,
		BlockCommitment:   *req.BlockHeader.BlockCommitment,
	}

	if err := a.SubmitWork(bh); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(true)
}

// GetWorkResp is resp struct for get-work API
type GetWorkResp struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
	Seed        *bc.Hash           `json:"seed"`
}

// GetWork gets work in compressed protobuf format
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

// GetWorkJSONResp is resp struct for get-work-json API
type GetWorkJSONResp struct {
	BlockHeader *BlockHeaderJSON `json:"block_header"`
	Seed        *bc.Hash         `json:"seed"`
}

// GetWorkJSON gets work in json format
func (a *API) GetWorkJSON() (*GetWorkJSONResp, error) {
	bh, err := a.miningPool.GetWork()
	if err != nil {
		return nil, err
	}

	seed, err := a.chain.CalcNextSeed(&bh.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	return &GetWorkJSONResp{
		BlockHeader: &BlockHeaderJSON{
			Version:           bh.Version,
			Height:            bh.Height,
			PreviousBlockHash: bh.PreviousBlockHash,
			Timestamp:         bh.Timestamp,
			Nonce:             bh.Nonce,
			Bits:              bh.Bits,
			BlockCommitment:   &bh.BlockCommitment,
		},
		Seed: seed,
	}, nil
}

// SubmitWork tries to submit work to the chain
func (a *API) SubmitWork(bh *types.BlockHeader) error {
	return a.miningPool.SubmitWork(bh)
}

func (a *API) setMining(in struct {
	IsMining bool `json:"is_mining"`
}) Response {
	if in.IsMining {
		if _, err := a.wallet.AccountMgr.GetMiningAddress(); err != nil {
			return NewErrorResponse(errors.New("Mining address does not exist"))
		}
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
