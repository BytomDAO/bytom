package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/testutil"
)

func TestBlock(t *testing.T) {
	cases := []struct {
		block *Block
		hex   string
		hash  bc.Hash
	}{
		{
			block: &Block{
				BlockHeader: BlockHeader{
					Version: 1,
					Height:  1,
				},
				Transactions: []*Tx{},
			},
			hex: strings.Join([]string{
				"03", // serialization flags
				"01", // version
				"01", // block height
				"0000000000000000000000000000000000000000000000000000000000000000", // prev block hash
				"00",   // timestamp
				"20",   // commitment extensible field length
				"0000000000000000000000000000000000000000000000000000000000000000", // transactions merkle root
				"0100", // block witness
				"0100", // sup links
				"00",   // num transactions
			}, ""),
			hash: testutil.MustDecodeHash("42e74d130e5ab27e8a71b90e7de8c8e00ecfa77456070202ab8509f7b0ab49ae"),
		},
		{
			block: &Block{
				BlockHeader: BlockHeader{
					Version:           1,
					Height:            432234,
					PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
					Timestamp:         1522908275,
					BlockCommitment: BlockCommitment{
						TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
					},
				},
				Transactions: []*Tx{
					NewTx(TxData{
						Version:        1,
						SerializedSize: uint64(284),
						TimeRange:      654,
						Inputs: []*TxInput{
							NewIssuanceInput([]byte("nonce"), 254354, []byte("issuanceProgram"), [][]byte{[]byte("arguments1"), []byte("arguments2")}, []byte("assetDefinition")),
							NewSpendInput([][]byte{[]byte("arguments3"), []byte("arguments4")}, testutil.MustDecodeHash("fad5195a0c8e3b590b86a3c0a95e7529565888508aecca96e9aeda633002f409"), *consensus.BTMAssetID, 254354, 3, []byte("spendProgram"), [][]byte{[]byte("stateData")}),
						},
						Outputs: []*TxOutput{
							NewOriginalTxOutput(testutil.MustDecodeAsset("a69849e11add96ac7053aad22ba2349a4abf5feb0475a0afcadff4e128be76cf"), 254354, []byte("true"), [][]byte{[]byte("stateData")}),
						},
					}),
					NewTx(TxData{
						Version:        1,
						SerializedSize: uint64(132),
						Inputs: []*TxInput{
							NewCoinbaseInput([]byte("arbitrary")),
						},
						Outputs: []*TxOutput{
							NewOriginalTxOutput(*consensus.BTMAssetID, 254354, []byte("true"), [][]byte{[]byte("stateData")}),
							NewOriginalTxOutput(*consensus.BTMAssetID, 254354, []byte("false"), [][]byte{[]byte("stateData")}),
						},
					}),
				},
			},
			hex: strings.Join([]string{
				"03",     // serialization flags
				"01",     // version
				"eab01a", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"f3f896d605", // timestamp
				"20",         // commitment extensible field length
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
				"0100", // block witness
				"0100", // sup links
				"02",   // num transactions
				"07018e0502012a00056e6f6e6365a69849e11add96ac7053aad22ba2349a4abf5feb0475a0afcadff4e128be76cf92c30f380f6173736574446566696e6974696f6e010f69737375616e636550726f6772616d020a617267756d656e7473310a617267756d656e747332015f015dfad5195a0c8e3b590b86a3c0a95e7529565888508aecca96e9aeda633002f409ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff92c30f03010c7370656e6450726f6772616d010973746174654461746117020a617267756d656e7473330a617267756d656e74733401010034a69849e11add96ac7053aad22ba2349a4abf5feb0475a0afcadff4e128be76cf92c30f010474727565010973746174654461746100",
				"07010001010b02096172626974726172790002010034ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff92c30f010474727565010973746174654461746100010035ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff92c30f010566616c7365010973746174654461746100",
			}, ""),
			hash: testutil.MustDecodeHash("6076fc8a96b08a4842f4bdc805606e9775ce6dbe4e371e88c70b75ea4283e942"),
		},
	}

	for i, test := range cases {
		got := testutil.Serialize(t, test.block)
		want, err := hex.DecodeString(test.hex)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(got, want) {
			t.Errorf("test %d: bytes = %x want %x", i, got, want)
		}

		blockHash := test.block.Hash()
		if blockHash != test.hash {
			t.Errorf("test %d: hash = %s want %s", i, blockHash.String(), test.hash.String())
		}

		blockJSON, err := json.Marshal(test.block)
		if err != nil {
			t.Errorf("test %d: error marshaling block to json: %s", i, err)
		}

		blockFromJSON := Block{}
		if err := json.Unmarshal(blockJSON, &blockFromJSON); err != nil {
			t.Errorf("test %d: error unmarshaling block from json: %s", i, err)
		}
		if !testutil.DeepEqual(*test.block, blockFromJSON) {
			t.Errorf("test %d: got:\n%s\nwant:\n%s", i, spew.Sdump(blockFromJSON), spew.Sdump(*test.block))
		}
	}
}

func TestReadFrom(t *testing.T) {
	btmAssetID := testutil.MustDecodeAsset("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	cases := []struct {
		rawBlock  string
		wantBlock Block
	}{
		{
			rawBlock: "03018b5f3077f24528e94ecfc4491bb2e9ed6264a632a9a4b86b00c88093ca545d14a137d4f5e1e4054035a2d11158f47a5c5267630b2b6cf9e9a5f79a598085a2572a68defeb8013ad26978a65b4ee5b6f4914fe5c05000459a803ecf59132604e5d334d64249c5e50a01000100020701000101080206003132313731000101003fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff809df3b49a010116001437e1aec83a4e6587ca9609e4e5aa728db70074490000070100020161015f4b5cb973f5bef4eadde4c89b92ee73312b940e84164da0594149554cc8a2adeaffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c480c1240201160014cb9f2391bafe2bc1159b2c4c8a0f17ba1b4dd94e006302405760b15cc09e543437c4e3aad05bf073e82ebdb214beccb5f4473653dfc0a9d5ae59fb149de19eb71c1c1399594757aeea4dd6327ca2790ef919bd20caa86104201381d35e235813ad1e62f9a602c82abee90565639cc4573568206b55bcd2aed90130000840142084606f20ca7b38dc897329a288ea31031724f5c55bcafec80468a546955023380af2faad1480d0dbc3f402b001467b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d0125ae2054a71277cc162eb3eb21b5bd9fe54402829a53b294deaed91692a2cd8a081f9c5151ad0140621c2c3554da50d2a492d9d78be7c6159359d8f5f0b93a054ce0133617a61d85c532aff449b97a3ec2804ca5fe12b4d54aa6e8c3215c33d04abee9c9abdfdb030201003effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c0d1e123011600144b61da45324299e40dacc255e2ea07dfce3a56d2000001003f7b38dc897329a288ea31031724f5c55bcafec80468a546955023380af2faad1480d0dbc3f4020116001437e1aec83a4e6587ca9609e4e5aa728db70074490000",
			wantBlock: Block{
				BlockHeader: BlockHeader{
					Version:           1,
					Height:            12171,
					PreviousBlockHash: testutil.MustDecodeHash("3077f24528e94ecfc4491bb2e9ed6264a632a9a4b86b00c88093ca545d14a137"),
					Timestamp:         1553496788,
					BlockCommitment: BlockCommitment{
						TransactionsMerkleRoot: testutil.MustDecodeHash("35a2d11158f47a5c5267630b2b6cf9e9a5f79a598085a2572a68defeb8013ad2"),
					},
				},
				Transactions: []*Tx{
					{
						TxData: TxData{
							Version:        1,
							SerializedSize: 83,
							TimeRange:      0,
							Inputs: []*TxInput{
								NewCoinbaseInput(testutil.MustDecodeHexString("003132313731")),
							},
							Outputs: []*TxOutput{
								NewOriginalTxOutput(btmAssetID, 41450000000, testutil.MustDecodeHexString("001437e1aec83a4e6587ca9609e4e5aa728db7007449"), nil),
							},
						},
					},
					{
						TxData: TxData{
							Version:        1,
							SerializedSize: 565,
							TimeRange:      0,
							Inputs: []*TxInput{
								NewSpendInput(
									[][]byte{
										testutil.MustDecodeHexString("5760b15cc09e543437c4e3aad05bf073e82ebdb214beccb5f4473653dfc0a9d5ae59fb149de19eb71c1c1399594757aeea4dd6327ca2790ef919bd20caa86104"),
										testutil.MustDecodeHexString("1381d35e235813ad1e62f9a602c82abee90565639cc4573568206b55bcd2aed9"),
									},
									testutil.MustDecodeHash("4b5cb973f5bef4eadde4c89b92ee73312b940e84164da0594149554cc8a2adea"),
									btmAssetID,
									9800000000,
									2,
									testutil.MustDecodeHexString("0014cb9f2391bafe2bc1159b2c4c8a0f17ba1b4dd94e"),
									nil,
								),
								NewIssuanceInput(
									testutil.MustDecodeHexString("40142084606f20ca"),
									100000000000,
									testutil.MustDecodeHexString("ae2054a71277cc162eb3eb21b5bd9fe54402829a53b294deaed91692a2cd8a081f9c5151ad"),
									[][]byte{testutil.MustDecodeHexString("621c2c3554da50d2a492d9d78be7c6159359d8f5f0b93a054ce0133617a61d85c532aff449b97a3ec2804ca5fe12b4d54aa6e8c3215c33d04abee9c9abdfdb03")},
									testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
								),
							},
							Outputs: []*TxOutput{
								NewOriginalTxOutput(btmAssetID, 9600000000, testutil.MustDecodeHexString("00144b61da45324299e40dacc255e2ea07dfce3a56d2"), nil),
								NewOriginalTxOutput(testutil.MustDecodeAsset("7b38dc897329a288ea31031724f5c55bcafec80468a546955023380af2faad14"), 100000000000, testutil.MustDecodeHexString("001437e1aec83a4e6587ca9609e4e5aa728db7007449"), nil),
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		blockBytes, err := hex.DecodeString(c.rawBlock)
		if err != nil {
			t.Fatal(err)
		}

		block := &Block{}
		if err := block.readFrom(blockchain.NewReader(blockBytes)); err != nil {
			t.Fatal(err)
		}

		for _, tx := range c.wantBlock.Transactions {
			tx.Tx = MapTx(&tx.TxData)
		}

		if !testutil.DeepEqual(*block, c.wantBlock) {
			t.Errorf("test block read from fail, got:%v, want:%v", *block, c.wantBlock)
		}
	}
}
