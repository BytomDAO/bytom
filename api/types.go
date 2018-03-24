package api

import (
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// BlockHeaderByHeight is resp struct for API
type BlockHeaderByHeight struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
	Reward      uint64             `json:"reward"`
}

// GetWorkResp is resp struct for API
type GetWorkResp struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
	Seed        *bc.Hash           `json:"seed"`
}

type NetInfo struct {
	Listening    bool   `json:"listening"`
	Syncing      bool   `json:"syncing"`
	Mining       bool   `json:"mining"`
	PeerCount    int    `json:"peer_count"`
	CurrentBlock uint64 `json:"current_block"`
	HighestBlock uint64 `json:"highest_block"`
}
