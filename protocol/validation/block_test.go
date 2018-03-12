package validation

import (
	"testing"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

func dummyValidateTx(*bc.Tx) error {
	return nil
}

func generate(tb testing.TB, prev *bc.Block) *bc.Block {
	b := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           1,
			Height:            prev.Height + 1,
			PreviousBlockHash: prev.ID,
			Timestamp:         prev.Timestamp + 1,
			BlockCommitment:   types.BlockCommitment{},
		},
	}

	var err error
	b.TransactionsMerkleRoot, err = bc.MerkleRoot(nil)
	if err != nil {
		tb.Fatal(err)
	}

	return types.MapBlock(b)
}
