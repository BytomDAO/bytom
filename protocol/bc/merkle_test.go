package bc_test

import (
	"testing"
	"time"

	. "github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
)

func TestMerkleRoot(t *testing.T) {
	cases := []struct {
		witnesses [][][]byte
		want      Hash
	}{{
		witnesses: [][][]byte{
			{
				{1},
				[]byte("00000"),
			},
		},
		want: mustDecodeHash("dd26282725467a18bf98ed14e022da9493436dd09372cfeae13080cbaaded00f"),
	}, {
		witnesses: [][][]byte{
			{
				{1},
				[]byte("000000"),
			},
			{
				{1},
				[]byte("111111"),
			},
		},
		want: mustDecodeHash("a112fb9aea40e6d8b2e9f443ec326d49bea7b61cd616b4bddeb9ebb010e76bf5"),
	}, {
		witnesses: [][][]byte{
			{
				{1},
				[]byte("000000"),
			},
			{
				{2},
				[]byte("111111"),
				[]byte("222222"),
			},
		},
		want: mustDecodeHash("a112fb9aea40e6d8b2e9f443ec326d49bea7b61cd616b4bddeb9ebb010e76bf5"),
	}}

	for _, c := range cases {
		var txs []*Tx
		for _, wit := range c.witnesses {
			txs = append(txs, types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					&types.TxInput{
						AssetVersion: 1,
						TypedInput: &types.SpendInput{
							Arguments: wit,
							SpendCommitment: types.SpendCommitment{
								AssetAmount: AssetAmount{
									AssetId: &AssetID{V0: 0},
								},
							},
						},
					},
				},
			}).Tx)
		}
		got, err := MerkleRoot(txs)
		if err != nil {
			t.Fatalf("unexpected error %s", err)
		}
		if got != c.want {
			t.Log("witnesses", c.witnesses)
			t.Errorf("got merkle root = %x want %x", got.Bytes(), c.want.Bytes())
		}
	}
}

func TestDuplicateLeaves(t *testing.T) {
	trueProg := []byte{byte(vm.OP_TRUE)}
	assetID := ComputeAssetID(trueProg, 1, &EmptyStringHash)
	txs := make([]*Tx, 6)
	for i := uint64(0); i < 6; i++ {
		now := []byte(time.Now().String())
		txs[i] = types.NewTx(types.TxData{
			Version: 1,
			Inputs:  []*types.TxInput{types.NewIssuanceInput(now, i, trueProg, nil, nil)},
			Outputs: []*types.TxOutput{types.NewTxOutput(assetID, i, trueProg)},
		}).Tx
	}

	// first, get the root of an unbalanced tree
	txns := []*Tx{txs[5], txs[4], txs[3], txs[2], txs[1], txs[0]}
	root1, err := MerkleRoot(txns)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	// now, get the root of a balanced tree that repeats leaves 0 and 1
	txns = []*Tx{txs[5], txs[4], txs[3], txs[2], txs[1], txs[0], txs[1], txs[0]}
	root2, err := MerkleRoot(txns)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	if root1 == root2 {
		t.Error("forged merkle tree by duplicating some leaves")
	}
}

func TestAllDuplicateLeaves(t *testing.T) {
	trueProg := []byte{byte(vm.OP_TRUE)}
	assetID := ComputeAssetID(trueProg, 1, &EmptyStringHash)
	now := []byte(time.Now().String())
	issuanceInp := types.NewIssuanceInput(now, 1, trueProg, nil, nil)

	tx := types.NewTx(types.TxData{
		Version: 1,
		Inputs:  []*types.TxInput{issuanceInp},
		Outputs: []*types.TxOutput{types.NewTxOutput(assetID, 1, trueProg)},
	}).Tx
	tx1, tx2, tx3, tx4, tx5, tx6 := tx, tx, tx, tx, tx, tx

	// first, get the root of an unbalanced tree
	txs := []*Tx{tx6, tx5, tx4, tx3, tx2, tx1}
	root1, err := MerkleRoot(txs)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	// now, get the root of a balanced tree that repeats leaves 5 and 6
	txs = []*Tx{tx6, tx5, tx6, tx5, tx4, tx3, tx2, tx1}
	root2, err := MerkleRoot(txs)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	if root1 == root2 {
		t.Error("forged merkle tree with all duplicate leaves")
	}
}

func mustDecodeHash(s string) (h Hash) {
	err := h.UnmarshalText([]byte(s))
	if err != nil {
		panic(err)
	}
	return h
}
