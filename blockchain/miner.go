package blockchain

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/protocol/bc/legacy"
)

// Get the parameters of mining
func (bcr *BlockchainReactor) getWork() *WorkResp {
	var resp WorkResp
	if block := bcr.mining.GetCurrentBlock(); block == nil {
		return nil
	} else {
		resp.Header = block.BlockHeader
	}

	return &resp
}

// Submit work for mining
func (bcr *BlockchainReactor) submitWork(header legacy.BlockHeader) Response {
	log.Infof("mining:---submitWork header:%v", header)
	bcr.mining.NotifySpawnBlock(header)
	return NewSuccessResponse(nil)
}

type WorkResp struct {
	Header legacy.BlockHeader
}
