package apinode

import (
	"encoding/json"

	"github.com/bytom/bytom/api"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc/types"
)

func (n *Node) GetBlockByHash(hash string) (*types.Block, error) {
	return n.getRawBlock(&getRawBlockReq{BlockHash: hash})
}

func (n *Node) GetBlockByHeight(height uint64) (*types.Block, error) {
	return n.getRawBlock(&getRawBlockReq{BlockHeight: height})
}

type getRawBlockReq struct {
	BlockHeight uint64 `json:"block_height"`
	BlockHash   string `json:"block_hash"`
}

func (n *Node) getRawBlock(req *getRawBlockReq) (*types.Block, error) {
	url := "/get-raw-block"
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}
	resp := &api.GetRawBlockResp{}
	return resp.RawBlock, n.request(url, payload, resp)
}

// bytomNetInfoResp is the response of bytom net info
type bytomNetInfoResp struct {
	FinalizedBlock uint64 `json:"finalized_block"`
}

// GetIrreversibleHeight return the irreversible block height of connected node
func (n *Node) GetIrreversibleHeight() (uint64, error) {
	url := "/net-info"
	res := &bytomNetInfoResp{}
	return res.FinalizedBlock, n.request(url, nil, res)
}
