package blockchain

import (
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// GetWorkResp is resp struct for API
type GetWorkResp struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
	Seed        *bc.Hash           `json:"seed"`
}

func (bcr *BlockchainReactor) GetWork() (*GetWorkResp, error) {
	bh, err := bcr.miningPool.GetWork()
	if err != nil {
		return nil, err
	}

	seed, err := bcr.chain.GetSeed(bh.Height, &bh.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	return &GetWorkResp{
		BlockHeader: bh,
		Seed:        seed,
	}, nil
}

func (bcr *BlockchainReactor) SubmitWork(bh *types.BlockHeader) bool {
	return bcr.miningPool.SubmitWork(bh)
}
