package types

import (
	"math/rand"
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
	cases := []struct {
		txCount          int
		relatedTxIndexes []int
		expectHashLen    int
		expectFlags      []uint8
	}{
		{
			txCount:          10,
			relatedTxIndexes: []int{0, 3, 7, 8},
			expectHashLen:    9,
			expectFlags:      []uint8{1, 1, 1, 1, 2, 0, 1, 0, 2, 1, 0, 1, 0, 2, 1, 2, 0},
		},
		{
			txCount:          1,
			relatedTxIndexes: []int{0},
			expectHashLen:    1,
			expectFlags:      []uint8{2},
		},
		{
			txCount:          19,
			relatedTxIndexes: []int{1, 3, 5, 7, 11, 15},
			expectHashLen:    15,
			expectFlags:      []uint8{1, 1, 1, 1, 1, 0, 2, 1, 0, 2, 1, 1, 0, 2, 1, 0, 2, 1, 1, 0, 1, 0, 2, 1, 0, 1, 0, 2, 0},
		},
	}
	for _, c := range cases {
		txs, bcTxs := mockTransactions(c.txCount)

		var nodes []merkleNode
		for _, tx := range txs {
			nodes = append(nodes, tx.ID)
		}
		tree := buildMerkleTree(nodes)
		root, err := TxMerkleRoot(bcTxs)
		if err != nil {
			t.Fatalf("unexpected error %s", err)
		}
		if tree.hash != root {
			t.Error("build tree fail")
		}

		var relatedTx []*Tx
		for _, index := range c.relatedTxIndexes {
			relatedTx = append(relatedTx, txs[index])
		}
		proofHashes, flags := GetTxMerkleTreeProof(txs, relatedTx)
		if !testutil.DeepEqual(flags, c.expectFlags) {
			t.Error("The flags is not equals expect flags", flags, c.expectFlags)
		}
		if len(proofHashes) != c.expectHashLen {
			t.Error("The length proof hashes is not equals expect length")
		}
		var ids []*bc.Hash
		for _, tx := range relatedTx {
			ids = append(ids, &tx.ID)
		}
		if !ValidateTxMerkleTreeProof(proofHashes, flags, ids, root) {
			t.Error("Merkle tree validate fail")
		}
	}
}

func TestStatusMerkleProof(t *testing.T) {
	cases := []struct {
		statusCount    int
		relatedIndexes []int
		flags          []uint8
		expectHashLen  int
	}{
		{
			statusCount:    10,
			relatedIndexes: []int{0, 3, 7, 8},
			flags:          []uint8{1, 1, 1, 1, 2, 0, 1, 0, 2, 1, 0, 1, 0, 2, 1, 2, 0},
			expectHashLen:  9,
		},
		{
			statusCount:    1,
			relatedIndexes: []int{0},
			flags:          []uint8{2},
			expectHashLen:  1,
		},
		{
			statusCount:    19,
			relatedIndexes: []int{1, 3, 5, 7, 11, 15},
			flags:          []uint8{1, 1, 1, 1, 1, 0, 2, 1, 0, 2, 1, 1, 0, 2, 1, 0, 2, 1, 1, 0, 1, 0, 2, 1, 0, 1, 0, 2, 0},
			expectHashLen:  15,
		},
	}
	for _, c := range cases {
		statuses := mockStatuses(c.statusCount)
		var relatedStatuses []*bc.TxVerifyResult
		for _, index := range c.relatedIndexes {
			relatedStatuses = append(relatedStatuses, statuses[index])
		}
		hashes := GetStatusMerkleTreeProof(statuses, c.flags)
		if len(hashes) != c.expectHashLen {
			t.Error("The length proof hashes is not equals expect length")
		}
		root, _ := TxStatusMerkleRoot(statuses)
		if !ValidateStatusMerkleTreeProof(hashes, c.flags, relatedStatuses, root) {
			t.Error("Merkle tree validate fail")
		}
	}
}

func TestUglyValidateTxMerkleProof(t *testing.T) {
	cases := []struct {
		hashes        [][32]byte
		flags         []uint8
		relatedHashes [][32]byte
		root          [32]byte
	}{
		{
			hashes:        [][32]byte{},
			flags:         []uint8{},
			relatedHashes: [][32]byte{},
			root:          [32]byte{},
		},
		{
			hashes:        [][32]byte{},
			flags:         []uint8{1, 1, 1, 1, 2, 0, 1, 0, 2, 1, 0, 1, 0, 2, 1, 2, 0},
			relatedHashes: [][32]byte{},
			root:          [32]byte{},
		},
		{
			hashes:        [][32]byte{},
			flags:         []uint8{1, 1, 1, 3, 2, 0, 5, 0, 2, 1, 2, 1, 0, 2, 1, 2, 0},
			relatedHashes: [][32]byte{},
			root:          [32]byte{},
		},
		{
			hashes: [][32]byte{
				{0, 147, 55, 10, 142, 25, 248, 241, 49, 253, 126, 117, 197, 118, 97, 89, 80, 213, 103, 46, 229, 225, 140, 99, 241, 5, 169, 91, 202, 180, 51, 44},
				{201, 183, 119, 152, 71, 251, 122, 183, 76, 244, 177, 231, 244, 85, 113, 51, 145, 143, 170, 43, 193, 48, 4, 39, 83, 65, 125, 251, 98, 177, 45, 250},
			},
			flags:         []uint8{},
			relatedHashes: [][32]byte{},
			root:          [32]byte{},
		},
		{
			hashes: [][32]byte{},
			flags:  []uint8{},
			relatedHashes: [][32]byte{
				{0, 147, 55, 10, 142, 25, 248, 241, 49, 253, 126, 117, 197, 118, 97, 89, 80, 213, 103, 46, 229, 225, 140, 99, 241, 5, 169, 91, 202, 180, 51, 44},
				{103, 218, 115, 138, 183, 208, 116, 2, 184, 29, 61, 136, 235, 37, 47, 96, 188, 58, 243, 180, 148, 86, 68, 212, 130, 145, 120, 148, 225, 155, 116, 234},
			},
			root: [32]byte{},
		},
		{
			hashes: [][32]byte{},
			flags:  []uint8{1, 1, 0, 2, 1, 2, 1, 0, 1},
			relatedHashes: [][32]byte{
				{0, 147, 55, 10, 142, 25, 248, 241, 49, 253, 126, 117, 197, 118, 97, 89, 80, 213, 103, 46, 229, 225, 140, 99, 241, 5, 169, 91, 202, 180, 51, 44},
				{103, 218, 115, 138, 183, 208, 116, 2, 184, 29, 61, 136, 235, 37, 47, 96, 188, 58, 243, 180, 148, 86, 68, 212, 130, 145, 120, 148, 225, 155, 116, 234},
			},
			root: [32]byte{40, 17, 56, 224, 169, 234, 25, 80, 88, 68, 189, 97, 162, 245, 132, 55, 135, 3, 87, 130, 192, 147, 218, 116, 209, 43, 95, 186, 115, 238, 235, 7},
		},
		{
			hashes: [][32]byte{
				{104, 240, 62, 162, 176, 42, 33, 173, 148, 77, 26, 67, 173, 97, 82, 167, 250, 106, 126, 212, 16, 29, 89, 190, 98, 89, 77, 211, 14, 242, 165, 88},
			},
			flags: []uint8{},
			relatedHashes: [][32]byte{
				{0, 147, 55, 10, 142, 25, 248, 241, 49, 253, 126, 117, 197, 118, 97, 89, 80, 213, 103, 46, 229, 225, 140, 99, 241, 5, 169, 91, 202, 180, 51, 44},
				{103, 218, 115, 138, 183, 208, 116, 2, 184, 29, 61, 136, 235, 37, 47, 96, 188, 58, 243, 180, 148, 86, 68, 212, 130, 145, 120, 148, 225, 155, 116, 234},
			},
			root: [32]byte{},
		},
	}

	for _, c := range cases {
		var hashes, relatedHashes []*bc.Hash
		for _, hashByte := range c.hashes {
			hash := bc.NewHash(hashByte)
			hashes = append(hashes, &hash)
		}
		for _, hashByte := range c.relatedHashes {
			hash := bc.NewHash(hashByte)
			relatedHashes = append(relatedHashes, &hash)
		}
		root := bc.NewHash(c.root)
		if ValidateTxMerkleTreeProof(hashes, c.flags, relatedHashes, root) != false {
			t.Error("Validate merkle tree proof fail")
		}
	}
}

func mockTransactions(txCount int) ([]*Tx, []*bc.Tx) {
	var txs []*Tx
	var bcTxs []*bc.Tx
	trueProg := []byte{byte(vm.OP_TRUE)}
	assetID := bc.ComputeAssetID(trueProg, 1, &bc.EmptyStringHash)
	for i := 0; i < txCount; i++ {
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
	return txs, bcTxs
}

func mockStatuses(statusCount int) []*bc.TxVerifyResult {
	var statuses []*bc.TxVerifyResult
	for i := 0; i < statusCount; i++ {
		status := &bc.TxVerifyResult{}
		fail := rand.Intn(2)
		if fail == 0 {
			status.StatusFail = true
		} else {
			status.StatusFail = false
		}
		statuses = append(statuses, status)
	}
	return statuses
}
