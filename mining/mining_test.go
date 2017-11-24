// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import (
	"fmt"
	"testing"
	"time"

	"github.com/bytom/consensus"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
)

func TestNewInitBlock(t *testing.T) {
	coinbaseTx, err := createCoinbaseTx(nil, 0, 1)
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

	var seed [32]byte
	sha3pool.Sum256(seed[:], make([]byte, 32))

	b := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:           1,
			Height:            1,
			PreviousBlockHash: bc.Hash{},
			TimestampMS:       bc.Millis(time.Now()),
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				AssetsMerkleRoot:       snap.Tree.RootHash(),
			},
			Bits: uint64(2161727821138738707),
			Seed: bc.NewHash(seed),
		},
		Transactions: []*legacy.Tx{coinbaseTx},
	}

	for i := uint64(0); i <= 10000000000000; i++ {
		b.Nonce = i
		hash := b.Hash()

		if consensus.CheckProofOfWork(&hash, b.Bits) {
			break
		}
	}

	rawBlock, err := b.MarshalText()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(rawBlock))
}
