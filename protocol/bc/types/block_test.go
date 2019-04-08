package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/consensus"
	"github.com/bytom/encoding/blockchain"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
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
				"00", // timestamp
				"40", // commitment extensible field length
				"0000000000000000000000000000000000000000000000000000000000000000", // transactions merkle root
				"0000000000000000000000000000000000000000000000000000000000000000", // tx status hash
				"00", // nonce
				"00", // bits
				"00", // num transactions
			}, ""),
			hash: testutil.MustDecodeHash("9609d2e45760f34cbc6c6d948c3fb9b6d7b61552d9d17fdd5b7d0cb5d2e67244"),
		},
		{
			block: &Block{
				BlockHeader: BlockHeader{
					Version:           1,
					Height:            432234,
					PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
					Timestamp:         1522908275,
					Nonce:             34342,
					Bits:              2305843009222082559,
					BlockCommitment: BlockCommitment{
						TransactionStatusHash:  testutil.MustDecodeHash("b94301ea4e316bee00109f68d25beaca90aeff08e9bf439a37d91d7a3b5a1470"),
						TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
					},
				},
				Transactions: []*Tx{
					NewTx(TxData{
						Version:        1,
						SerializedSize: uint64(261),
						TimeRange:      654,
						Inputs: []*TxInput{
							NewIssuanceInput([]byte("nonce"), 254354, []byte("issuanceProgram"), [][]byte{[]byte("arguments1"), []byte("arguments2")}, []byte("assetDefinition")),
							NewSpendInput([][]byte{[]byte("arguments3"), []byte("arguments4")}, testutil.MustDecodeHash("fad5195a0c8e3b590b86a3c0a95e7529565888508aecca96e9aeda633002f409"), *consensus.BTMAssetID, 254354, 3, []byte("spendProgram")),
						},
						Outputs: []*TxOutput{
							NewTxOutput(testutil.MustDecodeAsset("a69849e11add96ac7053aad22ba2349a4abf5feb0475a0afcadff4e128be76cf"), 254354, []byte("true")),
						},
					}),
					NewTx(TxData{
						Version:        1,
						SerializedSize: uint64(108),
						Inputs: []*TxInput{
							NewCoinbaseInput([]byte("arbitrary")),
						},
						Outputs: []*TxOutput{
							NewTxOutput(*consensus.BTMAssetID, 254354, []byte("true")),
							NewTxOutput(*consensus.BTMAssetID, 254354, []byte("false")),
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
				"40",         // commitment extensible field length
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
				"b94301ea4e316bee00109f68d25beaca90aeff08e9bf439a37d91d7a3b5a1470", // tx status hash
				"a68c02",             // nonce
				"ffffff838080808020", // bits
				"02",                 // num transactions
				"07018e0502012a00056e6f6e6365a69849e11add96ac7053aad22ba2349a4abf5feb0475a0afcadff4e128be76cf92c30f380f6173736574446566696e6974696f6e010f69737375616e636550726f6772616d020a617267756d656e7473310a617267756d656e74733201540152fad5195a0c8e3b590b86a3c0a95e7529565888508aecca96e9aeda633002f409ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff92c30f03010c7370656e6450726f6772616d17020a617267756d656e7473330a617267756d656e747334010129a69849e11add96ac7053aad22ba2349a4abf5feb0475a0afcadff4e128be76cf92c30f01047472756500",
				"07010001010b020961726269747261727900020129ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff92c30f01047472756500012affffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff92c30f010566616c736500",
			}, ""),
			hash: testutil.MustDecodeHash("86c833711a6a6b59864708d9dbae7869ba10782e3e7b1c7fc9fe3514899fec80"),
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
			rawBlock: "03018b5f3077f24528e94ecfc4491bb2e9ed6264a632a9a4b86b00c88093ca545d14a137d4f5e1e4054035a2d11158f47a5c5267630b2b6cf9e9a5f79a598085a2572a68defeb8013ad26978a65b4ee5b6f4914fe5c05000459a803ecf59132604e5d334d64249c5e50a17ebee908080808080200207010001010802060031323137310001013effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff809df3b49a010116001437e1aec83a4e6587ca9609e4e5aa728db700744900070100020160015e4b5cb973f5bef4eadde4c89b92ee73312b940e84164da0594149554cc8a2adeaffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c480c1240201160014cb9f2391bafe2bc1159b2c4c8a0f17ba1b4dd94e6302405760b15cc09e543437c4e3aad05bf073e82ebdb214beccb5f4473653dfc0a9d5ae59fb149de19eb71c1c1399594757aeea4dd6327ca2790ef919bd20caa86104201381d35e235813ad1e62f9a602c82abee90565639cc4573568206b55bcd2aed90130000840142084606f20ca7b38dc897329a288ea31031724f5c55bcafec80468a546955023380af2faad1480d0dbc3f402b001467b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d0125ae2054a71277cc162eb3eb21b5bd9fe54402829a53b294deaed91692a2cd8a081f9c5151ad0140621c2c3554da50d2a492d9d78be7c6159359d8f5f0b93a054ce0133617a61d85c532aff449b97a3ec2804ca5fe12b4d54aa6e8c3215c33d04abee9c9abdfdb0302013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c0d1e123011600144b61da45324299e40dacc255e2ea07dfce3a56d200013e7b38dc897329a288ea31031724f5c55bcafec80468a546955023380af2faad1480d0dbc3f4020116001437e1aec83a4e6587ca9609e4e5aa728db700744900",
			wantBlock: Block{
				BlockHeader: BlockHeader{
					Version:           1,
					Height:            12171,
					PreviousBlockHash: testutil.MustDecodeHash("3077f24528e94ecfc4491bb2e9ed6264a632a9a4b86b00c88093ca545d14a137"),
					Timestamp:         1553496788,
					Nonce:             23,
					Bits:              2305843009213970283,
					BlockCommitment: BlockCommitment{
						TransactionsMerkleRoot: testutil.MustDecodeHash("35a2d11158f47a5c5267630b2b6cf9e9a5f79a598085a2572a68defeb8013ad2"),
						TransactionStatusHash:  testutil.MustDecodeHash("6978a65b4ee5b6f4914fe5c05000459a803ecf59132604e5d334d64249c5e50a"),
					},
				},
				Transactions: []*Tx{
					{
						TxData: TxData{
							Version:        1,
							SerializedSize: 81,
							TimeRange:      0,
							Inputs: []*TxInput{
								NewCoinbaseInput(testutil.MustDecodeHexString("003132313731")),
							},
							Outputs: []*TxOutput{
								NewTxOutput(btmAssetID, 41450000000, testutil.MustDecodeHexString("001437e1aec83a4e6587ca9609e4e5aa728db7007449")),
							},
						},
					},
					{
						TxData: TxData{
							Version:        1,
							SerializedSize: 560,
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
								NewTxOutput(btmAssetID, 9600000000, testutil.MustDecodeHexString("00144b61da45324299e40dacc255e2ea07dfce3a56d2")),
								NewTxOutput(testutil.MustDecodeAsset("7b38dc897329a288ea31031724f5c55bcafec80468a546955023380af2faad14"), 100000000000, testutil.MustDecodeHexString("001437e1aec83a4e6587ca9609e4e5aa728db7007449")),
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
