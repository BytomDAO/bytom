package blockchain

import (
	"github.com/bytom/protocol/bc/legacy"
	log "github.com/sirupsen/logrus"
)

// Get the parameters of mining
func (bcr *BlockchainReactor) getWork() *WorkResp {
	var resp WorkResp
	if block := bcr.mining.GetCurrentBlock(); block == nil {
		return nil
	} else {
		resp.Header = block.BlockHeader
	}
	seedCaches := bcr.chain.SeedCaches()
	if seedCache, err := seedCaches.Get(&resp.Header.Seed); err != nil {
		return nil
	} else {
		resp.Cache = seedCache
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
	Cache  []uint32
}
