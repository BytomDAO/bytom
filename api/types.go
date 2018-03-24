package api

import (
	"github.com/bytom/protocol/bc/types"
)

// BlockHeaderByHeight is resp struct for API
type BlockHeaderByHeight struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
	Reward      uint64             `json:"reward"`
}
