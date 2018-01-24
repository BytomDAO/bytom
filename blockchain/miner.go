package blockchain

import (
	"github.com/bytom/mining"
	"github.com/bytom/protocol/bc/legacy"
)

// Get the parameters of mining
func (bcr *BlockchainReactor) getWork() *WorkResp {
	var resp WorkResp
	if block, err := mining.NewBlockTemplate(bcr.chain, bcr.txPool, bcr.accounts); err != nil {
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

type WorkResp struct {
	Header legacy.BlockHeader
	Cache  []uint32
}
