package blockchain

import (
	"github.com/bytom/mining"
	"github.com/bytom/protocol/bc/legacy"
)

// Get the parameters of mining
func (bcr *BlockchainReactor) getWork() Response {
	var resp workResp
	if block, err := mining.NewBlockTemplate(bcr.chain, bcr.txPool, bcr.accounts); err != nil {
		return NewErrorResponse(err)
	} else {
		resp.Header = block.BlockHeader
	}
	seedCaches := bcr.chain.SeedCaches()
	if seedCache, err := seedCaches.Get(&resp.Header.Seed); err != nil {
		return NewErrorResponse(err)
	} else {
		resp.cache = seedCache
	}
	return NewSuccessResponse(resp)
}

type workResp struct {
	Header legacy.BlockHeader `json:"header"`
	cache  []uint32       `json:"cache"`
}
