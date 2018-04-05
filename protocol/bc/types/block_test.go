package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/consensus"
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
