package blockchain

import (
	"context"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

type GetReward struct {
	Difficulty   uint64   `json:"difficulty"`
	Hash         *bc.Hash `json:"hash"`
	Height       uint64   `json:"height"`
	Version      uint64   `json:"version"`
	OrphanStatus bool     `json:"orphan_status"`
	PrevHash     *bc.Hash `json:"prev_hash"`
	Reward       uint64   `json:"reward "`
	Timestamp    uint64   `json:"timestamp"`
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
	Height uint64 `json:"block_height"`
}) Response {
	block, err := bcr.chain.GetBlockByHeight(req.Height)
	if err != nil {
		return NewErrorResponse(err)
	}

	hash := block.Hash()
	resp := &GetReward{
		Difficulty:   block.Bits,
		Hash:         &hash,
		Height:       block.Height,
		Version:      block.Version,
		OrphanStatus: false,
		PrevHash:     &block.PreviousBlockHash,
		Reward:       block.Transactions[0].Outputs[0].Amount,
		Timestamp:    block.Timestamp,
	}
	return NewSuccessResponse(resp)
}
