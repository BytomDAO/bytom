package types

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/testutil"
)

func TestMerkleRoot(t *testing.T) {
	cases := []struct {
		witnesses [][][]byte
		want      bc.Hash
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
		var txs []*bc.Tx
		for _, wit := range c.witnesses {
			txs = append(txs, NewTx(TxData{
				Inputs: []*TxInput{
					&TxInput{
						AssetVersion: 1,
						TypedInput: &SpendInput{
							Arguments: wit,
							SpendCommitment: SpendCommitment{
								AssetAmount: bc.AssetAmount{
									AssetId: &bc.AssetID{V0: 0},
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
	assetID := bc.ComputeAssetID(trueProg, 1, &bc.EmptyStringHash)
	txs := make([]*bc.Tx, 6)
	for i := uint64(0); i < 6; i++ {
		now := []byte(time.Now().String())
		txs[i] = NewTx(TxData{
			Version: 1,
			Inputs:  []*TxInput{NewIssuanceInput(now, i, trueProg, nil, nil)},
			Outputs: []*TxOutput{NewTxOutput(assetID, i, trueProg)},
		}).Tx
	}

	// first, get the root of an unbalanced tree
	txns := []*bc.Tx{txs[5], txs[4], txs[3], txs[2], txs[1], txs[0]}
	root1, err := TxMerkleRoot(txns)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	// now, get the root of a balanced tree that repeats leaves 0 and 1
	txns = []*bc.Tx{txs[5], txs[4], txs[3], txs[2], txs[1], txs[0], txs[1], txs[0]}
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
	assetID := bc.ComputeAssetID(trueProg, 1, &bc.EmptyStringHash)
	now := []byte(time.Now().String())
	issuanceInp := NewIssuanceInput(now, 1, trueProg, nil, nil)

	tx := NewTx(TxData{
		Version: 1,
		Inputs:  []*TxInput{issuanceInp},
		Outputs: []*TxOutput{NewTxOutput(assetID, 1, trueProg)},
	}).Tx
	tx1, tx2, tx3, tx4, tx5, tx6 := tx, tx, tx, tx, tx, tx

	// first, get the root of an unbalanced tree
	txs := []*bc.Tx{tx6, tx5, tx4, tx3, tx2, tx1}
	root1, err := TxMerkleRoot(txs)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	// now, get the root of a balanced tree that repeats leaves 5 and 6
	txs = []*bc.Tx{tx6, tx5, tx6, tx5, tx4, tx3, tx2, tx1}
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
	var bcTxs []*bc.Tx
	trueProg := []byte{byte(vm.OP_TRUE)}
	assetID := bc.ComputeAssetID(trueProg, 1, &bc.EmptyStringHash)
	for i := 0; i < 10; i++ {
		now := []byte(time.Now().String())
		issuanceInp := NewIssuanceInput(now, 1, trueProg, nil, nil)
		tx := NewTx(TxData{
			Version: 1,
			Inputs:  []*TxInput{issuanceInp},
			Outputs: []*TxOutput{NewTxOutput(assetID, 1, trueProg)},
		})
		txs = append(txs, tx)
		bcTxs = append(bcTxs, tx.Tx)
	}
	root, err := TxMerkleRoot(bcTxs)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	relatedTx := []*Tx{txs[0], txs[3], txs[7], txs[8]}
	proofHashes, flags := GetTxMerkleTreeProof(txs, relatedTx)
	if len(proofHashes) <= 0 {
		t.Error("Can not find any tx id in the merkle tree")
	}
	expectFlags := []uint8{1, 1, 1, 1, 2, 0, 1, 0, 2, 1, 0, 1, 0, 2, 1, 2, 0}
	if !reflect.DeepEqual(flags, expectFlags) {
		t.Error("The flags is not equals expect flags", flags)
	}
	if len(proofHashes) != 9 {
		t.Error("The length proof hashes is not equals expect length")
	}
	ids := []*bc.Hash{&txs[0].ID, &txs[3].ID, &txs[7].ID, &txs[8].ID}
	if !ValidateTxMerkleTreeProof(proofHashes, flags, ids, root) {
		t.Error("Merkle tree validate fail")
	}
}

func TestStatusMerkleProof(t *testing.T) {
	var statuses []*bc.TxVerifyResult
	for i := 0; i < 10; i++ {
		status := &bc.TxVerifyResult{}
		fail := rand.Intn(2)
		if fail == 0 {
			status.StatusFail = true
		} else {
			status.StatusFail = false
		}
		statuses = append(statuses, status)
	}
	relatedStatuses := []*bc.TxVerifyResult{statuses[0], statuses[3], statuses[7], statuses[8]}
	flags := []uint8{1, 1, 1, 1, 2, 0, 1, 0, 2, 1, 0, 1, 0, 2, 1, 2, 0}
	hashes := GetStatusMerkleTreeProof(statuses, flags)
	if len(hashes) != 9 {
		t.Error("The length proof hashes is not equals expect length")
	}
	root, _ := TxStatusMerkleRoot(statuses)
	if !ValidateStatusMerkleTreeProof(hashes, flags, relatedStatuses, root) {
		t.Error("Merkle tree validate fail")
	}
}
