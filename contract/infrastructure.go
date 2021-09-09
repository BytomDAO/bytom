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
	BestBlockHeight() uint64
	BestBlockHash() *bc.Hash
}

type Repository interface {
	GetBlock(hash *bc.Hash) (*types.Block, error)
	GetBlockByHeight(height uint64) (*types.Block, error)
	GetInstance(traceID string) (*Instance, error)
	LoadInstances() ([]*Instance, error)
	SaveInstances(instances []*Instance) error
	RemoveInstance(traceID string)
}
