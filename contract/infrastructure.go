package contract

import (
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

type Infrastructure struct {
	Chain      ChainService
	Repository Repository
}

func NewInfrastructure(chain ChainService, repository Repository) *Infrastructure {
	return &Infrastructure{Chain: chain, Repository: repository}
}

type ChainService interface {
	BestChain() (uint64, bc.Hash)
	FinalizedHeight() uint64
	GetBlockByHash(*bc.Hash) (*types.Block, error)
	GetBlockByHeight(uint64) (*types.Block, error)
	BlockWaiter(height uint64) <-chan struct{}
}

type ChainStatus struct {
	BlockHeight uint64  `json:"block_height"`
	BlockHash   bc.Hash `json:"block_hash"`
}

type Repository interface {
	GetInstance(traceID string) (*Instance, error)
	LoadInstances() ([]*Instance, error)
	SaveInstances(instances []*Instance) error
	SaveInstancesWithStatus(instances []*Instance, blockHeight uint64, blockHash bc.Hash) error
	RemoveInstance(traceID string)
	GetChainStatus() *ChainStatus
	SaveChainStatus(status *ChainStatus) error
}
