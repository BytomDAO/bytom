package types

import (
	"encoding/hex"

	"math/rand"
	"testing"
	"time"

	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/vm"
	"github.com/bytom/bytom/testutil"
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

func TestMerkleRootRealTx(t *testing.T) {
	rawTxs := []string{
		"070100010160015e5ac79a73db78e5c9215b37cb752f0147d1157c542bb4884908ceb97abc33fe0affffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa0f280d42b0001160014085a02ecdf934a56343aa59a3dec9d9feb86ee43630240035e1ef422b4901997ad3c20c50d82e726d03cb6e8ccb5dddc20e0c09e0a6f2e0055331e2b54d9ec52cffb1c47d8fdf2f8887d55c336753637cbf8f832c7af0b20a29601468f08c57ca9c383d28736a9d5c7737cd483126d8db3d85490fe497b3502013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa0aad1b30601160014991b78d1bf731390e2dd838c05ff37ec5146886b00013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8086d8f024011600145ade29df622cc68d0473aa1a20fb89690451c66e00",
		"070100020160015e4b5cb973f5bef4eadde4c89b92ee73312b940e84164da0594149554cc8a2adeaffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c480c1240201160014cb9f2391bafe2bc1159b2c4c8a0f17ba1b4dd94e630240d96b8f31519c5e34ef983bb7dfb92e807df7fc1ae5a4c08846d00d4f84ebd2f8634b9e0b0374eb2508d0f989520f622aef051862c26daba0e466944e3d55d00b201381d35e235813ad1e62f9a602c82abee90565639cc4573568206b55bcd2aed901300008ede605460cacbf107b38dc897329a288ea31031724f5c55bcafec80468a546955023380af2faad1480d0dbc3f402b001467b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d0125ae2054a71277cc162eb3eb21b5bd9fe54402829a53b294deaed91692a2cd8a081f9c5151ad01403a54a3ca0210d005cc9bce490478b518c405ba72e0bc1d134b739f29a73e008345229f0e061c420aa3c56a48bc1c9bf592914252ab9100e69252deeac532430f03013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80e0e8f011011600144ab5249140ca4630729030941f59f75e507bd4d500013e7b38dc897329a288ea31031724f5c55bcafec80468a546955023380af2faad1480d0dbc3f402011600145ade29df622cc68d0473aa1a20fb89690451c66e00013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80a2c0a012011600145ade29df622cc68d0473aa1a20fb89690451c66e00",
		"07010001016c016acf24f1471d67c25a01ac84482ecdd8550229180171cae22321f87fe43d4f6a13ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80b4c4c32101012200200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66ef02044059c7a12d006fd34bf8b9b2cb2f99756e5c3c3fdca4c928b830c014819e933b01c92a99bfeb6add73a5087870a3de3465cfed2c99f736b5f77d5fbdc69d91ff0040b95d110d118b873a8232104a6613f0e8c6a791efa3a695c02108cebd5239c8a8471551a48f18ab8ea05d10900b485af5e95b74cd3c01044c1742e71854099c0b40a1b63dae273e3b5b757b7c61286088a934e7282e837d08d62e60d7f75eb739529cd8c6cfef2254d47a546bf8b789657ce0944fec2f7e130c8498e28cae2a9108a901ae20d441b6f375659325a04eede4fc3b74579bb08ccd05b41b99776501e22d6dca7320af6d98ca2c3cd10bf0affbfa6e86609b750523cfadb662ec963c164f05798a49209820b9f1553b03aaebe7e3f9e9222ed7db73b5079b18564042fd3b2cef74156a20271b52de5f554aa4a6f1358f1c2193617bfb3fed4546d13c4af773096a429f9420eeb4a78d8b5cb8283c221ca2d3fd96b8946b3cddee02b7ceffb8f605932588595355ad020149ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80a0d9e61d012200206e8060ef3daca62841802dd9660b24b7dca81c1662b2d68ba8884ecbcd3e1e2200013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80d293ad0301160014ed7d3c466dbc6cc1f3a9af21267ac162f11b30a200",
		"070100020161015f4b5cb973f5bef4eadde4c89b92ee73312b940e84164da0594149554cc8a2adea0dafd0f0e42f06f3bf9a8cf5787519d3860650f27a2b3393d34e1fe06e89b469ddc3f8c2f40200011600141da7f908979e521bf2ba12d280b2c84fc1d024416302409524d0d817176eeb718ce45671d95831cdb138d27289aa8a920104e38a8cab8a7dc8cc3fb60d65aa337b719aed0f696fb12610bfe68add89169a47ac1241e0002033444e1b57524161af3899e50fdfe270a90a1ea97fe38e86019a1e252667fb2d0161015fed3181c99ca80db720231aee6948e1183bfe29c64208c1769afa7f938d3b2cf0ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff809cd2b0f4020101160014cfbccfac5018ad4b4bfbcb1fab834e3c8503746063024065beb1da2f0840188af0e3c0127b158f7a2a36f1612499694a731df1e3a9d1abe6694c42986b8700aa9856f59cb3692ee88d68b20d1278f05592fb253c58bd0520e5966eee4092eeefdd805b06f2ad368bb9392edec20998993ebe2a929052c1ce03013e0dafd0f0e42f06f3bf9a8cf5787519d3860650f27a2b3393d34e1fe06e89b469ddfbc8a2cf0201160014583c0323603dd397ba5414255adc80b076cf232b00013d0dafd0f0e42f06f3bf9a8cf5787519d3860650f27a2b3393d34e1fe06e89b46980c8afa02501160014fdb3e6abf7f430fdabb53484ca2469103b2af1b500013effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80dafa80f4020116001408e75789f47d2a39622e5a940fa918260bf44c5400",
		"07010001016d016b1f134a47da4f6df00822935e02a07514718ea99ce5ac4e07bd6c204e098eb525ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808a858fa70200012200206205ec178dc1ac6ea05ea01bb0fcda6aa978173026fa75204a101bdad7bd6b4889010240d8d5bbf4969fba52df8fba06f75c5de0f51b2bd5f902bf234591f90e78bae20bfb5b7904cb83a1d6577c431f644d37722b432df9d64718b8300e3ab74a871a0046ae2068003e53d467b6d81beaf1e7bd9b60a5ffedc79b36ce14ecd1f30a2dcbcd0551200449030407a3a1fa0731f7f784a72c325b5ce4d534fc3cf8fb7140536ba928605152ad02014affffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80f699b2a302012200209a0b4b27fde7d29d3b465d20eb2e19f4bda3a873d19d11f4cba53958bde92ed000013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80b3ffc40301160014ed7d3c466dbc6cc1f3a9af21267ac162f11b30a200",
	}
	wantMerkleRoot := "0f07b8a453771c2dc628f3895ebb33fea82a8de42e11aa588bec26419af22065"

	var txs []*bc.Tx
	for _, rawTx := range rawTxs {
		tx := Tx{}		
		if err := tx.UnmarshalText([]byte(rawTx)); err != nil {
			t.Fatal(err)
		}

		txs = append(txs, tx.Tx)
	}

	gotMerkleRoot, err := TxMerkleRoot(txs)
	if err != nil {
		t.Fatal(err)
	}

	if wantMerkleRoot != gotMerkleRoot.String() {
		t.Errorf("got merkle root:%s, want merkle root:%s", gotMerkleRoot.String(), wantMerkleRoot)
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
			txCount:          10,
			relatedTxIndexes: []int{},
			expectHashLen:    1,
			expectFlags:      []uint8{0},
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
			statusCount:    10,
			relatedIndexes: []int{},
			flags:          []uint8{0},
			expectHashLen:  1,
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
		hashes        []string
		flags         []uint8
		relatedHashes []string
		root          string
		expectResult  bool
	}{
		{
			hashes:        []string{},
			flags:         []uint8{},
			relatedHashes: []string{},
			root:          "",
			expectResult:  false,
		},
		{
			hashes:        []string{},
			flags:         []uint8{1, 1, 1, 1, 2, 0, 1, 0, 2, 1, 0, 1, 0, 2, 1, 2, 0},
			relatedHashes: []string{},
			root:          "",
			expectResult:  false,
		},
		{
			hashes: []string{
				"0093370a8e19f8f131fd7e75c576615950d5672ee5e18c63f105a95bcab4332c",
				"c9b7779847fb7ab74cf4b1e7f4557133918faa2bc130042753417dfb62b12dfa",
			},
			flags:         []uint8{},
			relatedHashes: []string{},
			root:          "",
			expectResult:  false,
		},
		{
			hashes: []string{},
			flags:  []uint8{},
			relatedHashes: []string{
				"0093370a8e19f8f131fd7e75c576615950d5672ee5e18c63f105a95bcab4332c",
				"c9b7779847fb7ab74cf4b1e7f4557133918faa2bc130042753417dfb62b12dfa",
			},
			root:         "",
			expectResult: false,
		},
		{
			hashes: []string{},
			flags:  []uint8{1, 1, 0, 2, 1, 2, 1, 0, 1},
			relatedHashes: []string{
				"0093370a8e19f8f131fd7e75c576615950d5672ee5e18c63f105a95bcab4332c",
				"c9b7779847fb7ab74cf4b1e7f4557133918faa2bc130042753417dfb62b12dfa",
			},
			root: "281138e0a9ea19505844bd61a2f5843787035782c093da74d12b5fba73eeeb07",
		},
		{
			hashes: []string{
				"68f03ea2b02a21ad944d1a43ad6152a7fa6a7ed4101d59be62594dd30ef2a558",
			},
			flags: []uint8{},
			relatedHashes: []string{
				"0093370a8e19f8f131fd7e75c576615950d5672ee5e18c63f105a95bcab4332c",
				"c9b7779847fb7ab74cf4b1e7f4557133918faa2bc130042753417dfb62b12dfa",
			},
			root:         "281138e0a9ea19505844bd61a2f5843787035782c093da74d12b5fba73eeeb07",
			expectResult: false,
		},
		{
			hashes: []string{
				"8ec3ee7589f95eee9b534f71fcd37142bcc839a0dbfe78124df9663827b90c35",
				"011bd3380852b2946df507e0c6234222c559eec8f545e4bc58a89e960892259b",
				"c205988d9c864083421f1bdb95e6cf8b52070facfcc87e46a6e8197f5389fca2",
			},
			flags: []uint8{1, 1, 0, 2, 0},
			relatedHashes: []string{
				"504af455e328e7dd39bbc059529851946d54ee8b459b11b3aac4a0feeb474487",
			},
			root:         "aff81a46fe79204ef9007243f374d54104a59762b9f74d80d56b5291753db6fb",
			expectResult: true,
		},
		// flags and hashes is correct, but relatedHashes has hash that does not exist
		{
			hashes: []string{
				"8ec3ee7589f95eee9b534f71fcd37142bcc839a0dbfe78124df9663827b90c35",
				"011bd3380852b2946df507e0c6234222c559eec8f545e4bc58a89e960892259b",
				"c205988d9c864083421f1bdb95e6cf8b52070facfcc87e46a6e8197f5389fca2",
			},
			flags: []uint8{1, 1, 0, 2, 0},
			relatedHashes: []string{
				"504af455e328e7dd39bbc059529851946d54ee8b459b11b3aac4a0feeb474487",
				"281138e0a9ea19505844bd61a2f5843787035782c093da74d12b5fba73eeeb07",
			},
			root:         "aff81a46fe79204ef9007243f374d54104a59762b9f74d80d56b5291753db6fb",
			expectResult: false,
		},
		// flags and hashes is correct, but relatedHashes is not enough
		{
			hashes: []string{
				"8ec3ee7589f95eee9b534f71fcd37142bcc839a0dbfe78124df9663827b90c35",
				"011bd3380852b2946df507e0c6234222c559eec8f545e4bc58a89e960892259b",
				"c205988d9c864083421f1bdb95e6cf8b52070facfcc87e46a6e8197f5389fca2",
			},
			flags:         []uint8{1, 1, 0, 2, 0},
			relatedHashes: []string{},
			root:          "aff81a46fe79204ef9007243f374d54104a59762b9f74d80d56b5291753db6fb",
			expectResult:  false,
		},
		// flags is correct, but hashes has additional hash at the end
		{
			hashes: []string{
				"8ec3ee7589f95eee9b534f71fcd37142bcc839a0dbfe78124df9663827b90c35",
				"011bd3380852b2946df507e0c6234222c559eec8f545e4bc58a89e960892259b",
				"c205988d9c864083421f1bdb95e6cf8b52070facfcc87e46a6e8197f5389fca2",
				"5a06c90136e81c0f9cad29725e69edc6d21bd6fb0641265f9c4b6bb6840b37dd",
			},
			flags: []uint8{1, 1, 0, 2, 0},
			relatedHashes: []string{
				"504af455e328e7dd39bbc059529851946d54ee8b459b11b3aac4a0feeb474487",
			},
			root:         "aff81a46fe79204ef9007243f374d54104a59762b9f74d80d56b5291753db6fb",
			expectResult: true,
		},
	}

	for _, c := range cases {
		var hashes, relatedHashes []*bc.Hash
		var hashBytes, rootBytes [32]byte
		var err error
		for _, hashStr := range c.hashes {
			if hashBytes, err = convertHashStr2Bytes(hashStr); err != nil {
				t.Fatal(err)
			}

			hash := bc.NewHash(hashBytes)
			hashes = append(hashes, &hash)
		}
		for _, hashStr := range c.relatedHashes {
			if hashBytes, err = convertHashStr2Bytes(hashStr); err != nil {
				t.Fatal(err)
			}

			hash := bc.NewHash(hashBytes)
			relatedHashes = append(relatedHashes, &hash)
		}
		if rootBytes, err = convertHashStr2Bytes(c.root); err != nil {
			t.Fatal(err)
		}

		root := bc.NewHash(rootBytes)
		if ValidateTxMerkleTreeProof(hashes, c.flags, relatedHashes, root) != c.expectResult {
			t.Error("Validate merkle tree proof fail")
		}
	}
}

func convertHashStr2Bytes(hashStr string) ([32]byte, error) {
	var result [32]byte
	hashBytes, err := hex.DecodeString(hashStr)
	if err != nil {
		return result, err
	}
	copy(result[:], hashBytes)
	return result, nil
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
