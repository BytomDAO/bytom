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
	GetBlock(hash bc.Hash) (*types.Block, error)
	GetBlockByHeight(height uint64) (*types.Block, error)
}

type Repository interface {
	GetInstance(id string) (*Instance, error)
	LoadInstances() ([]*Instance, error)
	SaveInstances(instances []*Instance) error
	RemoveInstance(id string) error
}
