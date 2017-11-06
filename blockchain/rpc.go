package blockchain

import (
	"context"
	//	"encoding/json"
	//	"net/http"

	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	//	"github.com/bytom/net/http/httpjson"
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

/*
func (a *BlockchainReactor) getSnapshotInfoRPC(ctx context.Context) (resp snapshotInfoResp, err error) {
	// TODO(jackson): cache latest snapshot and its height & size in-memory.
	resp.Height, resp.Size, err = a.store.LatestSnapshotInfo(ctx)
	resp.BlockchainID = *a.config.BlockchainId
	return resp, err
}

// getSnapshotRPC returns the raw protobuf snapshot at the provided height.
// Non-generators can call this endpoint to get raw data
// that they can use to populate their own snapshot table.
//
// This handler doesn't use the httpjson.Handler format so that it can return
// raw protobuf bytes on the wire.
func (a *BlockchainReactor) getSnapshotRPC(rw http.ResponseWriter, req *http.Request) {
	if a.config == nil {
		alwaysError(errUnconfigured).ServeHTTP(rw, req)
		return
	}

	var height uint64
	err := json.NewDecoder(req.Body).Decode(&height)
	if err != nil {
		errorFormatter.Write(req.Context(), rw, httpjson.ErrBadRequest)
		return
	}

	data, err := a.store.GetSnapshot(req.Context(), height)
	if err != nil {
		errorFormatter.Write(req.Context(), rw, err)
		return
	}
	rw.Header().Set("Content-Type", "application/x-protobuf")
	rw.Write(data)
}*/
