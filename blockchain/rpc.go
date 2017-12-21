package blockchain

import (
	"context"

	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

// getBlockRPC returns the block at the requested height.
// If successful, it always returns at least one block,
// waiting if necessary until one is created.
// It is an error to request blocks very far in the future.
func (a *BlockchainReactor) getBlockRPC(ctx context.Context, height uint64) (chainjson.HexBytes, error) {
	err := <-a.chain.BlockSoonWaiter(ctx, height)
	if err != nil {
		return nil, errors.Wrapf(err, "waiting for block at height %d", height)
	}

	block, err := a.chain.GetBlockByHeight(height)
	if err != nil {
		return nil, err
	}
	rawBlock, err := block.MarshalText()
	if err != nil {

		return nil, err
	}

	return rawBlock, nil
}

type snapshotInfoResp struct {
	Height       uint64  `json:"height"`
	Size         uint64  `json:"size"`
	BlockchainID bc.Hash `json:"blockchain_id"`
}
