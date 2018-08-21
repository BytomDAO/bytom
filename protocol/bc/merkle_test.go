package bc_test

import (
	"testing"
	"time"
	"math/rand"
	"reflect"

	. "github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/testutil"
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
		want: testutil.MustDecodeHash("fe34dbd5da0ce3656f423fd7aad7fc7e879353174d33a6446c2ed0e3f3512101"),
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
		want: testutil.MustDecodeHash("0e4b4c1af18b8f59997804d69f8f66879ad5e30027346ee003ff7c7a512e5554"),
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
		want: testutil.MustDecodeHash("0e4b4c1af18b8f59997804d69f8f66879ad5e30027346ee003ff7c7a512e5554"),
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
		got, err := TxMerkleRoot(txs)
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
	root1, err := TxMerkleRoot(txns)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	// now, get the root of a balanced tree that repeats leaves 0 and 1
	txns = []*Tx{txs[5], txs[4], txs[3], txs[2], txs[1], txs[0], txs[1], txs[0]}
	root2, err := TxMerkleRoot(txns)
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
	root1, err := TxMerkleRoot(txs)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	// now, get the root of a balanced tree that repeats leaves 5 and 6
	txs = []*Tx{tx6, tx5, tx6, tx5, tx4, tx3, tx2, tx1}
	root2, err := TxMerkleRoot(txs)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	if root1 == root2 {
		t.Error("forged merkle tree with all duplicate leaves")
	}
}

func TestTxMerkleProof(t *testing.T) {
	var txs []*Tx
	trueProg := []byte{byte(vm.OP_TRUE)}
	assetID := ComputeAssetID(trueProg, 1, &EmptyStringHash)
	for i := 0; i < 10; i++ {
		now := []byte(time.Now().String())
		issuanceInp := types.NewIssuanceInput(now, 1, trueProg, nil, nil)
		tx := types.NewTx(types.TxData{
			Version: 1,
			Inputs:  []*types.TxInput{issuanceInp},
			Outputs: []*types.TxOutput{types.NewTxOutput(assetID, 1, trueProg)},
		}).Tx
		txs = append(txs, tx)
	}
	root, err := TxMerkleRoot(txs)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	var txIDs []Hash
	var nodes []MerkleNode
	for _, tx := range txs {
		txIDs = append(txIDs, tx.ID)
		nodes = append(nodes, &tx.ID)
	}
	tree := BuildMerkleTree(nodes)
	if *tree.MerkleHash != root {
		t.Fatal(err)
	}

	relatedTx := []Hash{txs[0].ID, txs[3].ID, txs[7].ID, txs[8].ID}
	find, proofHashes, flags := GetTxMerkleTreeProof(txIDs, relatedTx)
	if !find {
		t.Error("Can not find any tx id in the merkle tree")
	}
	expectFlags := []uint8{1, 1, 1, 1, 2, 0, 1, 0, 2, 1, 0, 1, 0, 2, 1, 2, 0}
	if !reflect.DeepEqual(flags, expectFlags) {
		t.Error("The flags is not equals expect flags")
	}
	if len(proofHashes) != 9 {
		t.Error("The length proof hashes is not equals expect length")
	}
	ids := []Hash{txs[0].ID, txs[3].ID, txs[7].ID, txs[8].ID}
	if !ValidateTxMerkleTreeProof(proofHashes, flags, ids, root) {
		t.Error("Merkle tree validate fail")
	}
}

func TestStatusMerkleProof(t *testing.T) {
	var statuses []*TxVerifyResult
	for i := 0; i < 10; i++ {
		status := &TxVerifyResult{}
		fail := rand.Intn(2)
		if fail == 0 {
			status.StatusFail = true
		} else {
			status.StatusFail = false
		}
		statuses = append(statuses, status)
	}
	relatedStatuses := []*TxVerifyResult{statuses[3], statuses[6], statuses[8]}
	find, hashes, flags := GetStatusMerkleTreeProof(statuses, relatedStatuses)
	if !find {
		t.Error("Can not find any status in the merkle tree")
	}
	root, _ := TxStatusMerkleRoot(statuses)
	if !ValidateStatusMerkleTreeProof(hashes, flags, relatedStatuses, root) {
		t.Error("Merkle tree validate fail")
	}
}

func TestMerkleFlag(t *testing.T) {
	var flags []uint8
	flags = append(flags, FlagAssist)
	flags = append(flags, FlagTxParent)
	flags = append(flags, FlagTxParent)
	flags = append(flags, FlagTxLeaf)
	merkleFlags := NewMerkleFlags(flags)
	bytes := merkleFlags.Bytes()
	if bytes[0] != 0x68 {
		t.Error("Invalid merkle flags", bytes[0])
	}
}
