// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import (
	"fmt"
	"testing"
	"time"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
)

func TestNewInitBlock(t *testing.T) {
	coinbaseTx, err := createCoinbaseTx(0, 0, []byte{})
	if err != nil {
		t.Error(err)
	}
	merkleRoot, err := bc.MerkleRoot([]*bc.Tx{coinbaseTx.Tx})
	if err != nil {
		t.Error(err)
	}
	snap := state.Empty()
	if err := snap.ApplyTx(coinbaseTx.Tx); err != nil {
		t.Error(err)
	}

	b := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:           1,
			Height:            0,
			PreviousBlockHash: bc.Hash{},
			TimestampMS:       bc.Millis(time.Now()),
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				AssetsMerkleRoot:       snap.Tree.RootHash(),
			},
			Bits:  uint64(21617278211387387),
			Nonce: 0,
		},
		Transactions: []*legacy.Tx{coinbaseTx},
	}

	rawBlock, err := b.MarshalText()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(rawBlock))
}
