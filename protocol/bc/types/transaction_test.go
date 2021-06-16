package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/testutil"
)

func TestTransaction(t *testing.T) {
	cases := []struct {
		tx   *Tx
		hex  string
		hash bc.Hash
	}{
		{
			tx: NewTx(TxData{
				Version:        1,
				SerializedSize: uint64(5),
				Inputs:         nil,
				Outputs:        nil,
			}),
			hex: strings.Join([]string{
				"07", // serflags
				"01", // transaction version
				"00", // tx time range
				"00", // inputs count
				"00", // outputs count
			}, ""),
			hash: testutil.MustDecodeHash("8e88b9cb4615128c7209dff695f68b8de5b38648bf3d44d2d0e6a674848539c9"),
		},
		{
			tx: NewTx(TxData{
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
			hex: strings.Join([]string{
				"07",         // serflags
				"01",         // transaction version
				"8e05",       // tx time range
				"02",         // inputs count
				"01",         // input 0: asset version
				"2a",         // input 0: serialization length
				"00",         // input 0: issuance type flag
				"05",         // input 0: nonce length
				"6e6f6e6365", // input 0: nonce
				"a69849e11add96ac7053aad22ba2349a4abf5feb0475a0afcadff4e128be76cf", // input 0: assetID
				"92c30f",                         // input 0: amount
				"38",                             // input 0: input witness length
				"0f",                             // input 0: asset definition length
				"6173736574446566696e6974696f6e", // input 0: asset definition
				"01",                             // input 0: vm version
				"0f",                             // input 0: issuanceProgram length
				"69737375616e636550726f6772616d", // input 0: issuance program
				"02",                             // input 0: argument array length
				"0a",                             // input 0: first argument length
				"617267756d656e747331",           // input 0: first argument data
				"0a",                             // input 0: second argument length
				"617267756d656e747332",           // input 0: second argument data
				"01",                             // input 1: asset version
				"5f",                             // input 1: input commitment length
				"01",                             // input 1: spend type flag
				"5d",                             // input 1: spend commitment length
				"fad5195a0c8e3b590b86a3c0a95e7529565888508aecca96e9aeda633002f409", // input 1: source id
				"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", // input 1: assetID
				"92c30f",                   // input 1: amount
				"03",                       // input 1: source position
				"01",                       // input 1: vm version
				"0c",                       // input 1: spend program length
				"7370656e6450726f6772616d", // input 1: spend program
				"0109",                     // input 1: state data length
				"737461746544617461",       // input 1: state data
				"17",                       // input 1: witness length
				"02",                       // input 1: argument array length
				"0a",                       // input 1: first argument length
				"617267756d656e747333",     // input 1: first argument data
				"0a",                       // input 1: second argument length
				"617267756d656e747334",     // input 1: second argument data
				"01",                       // outputs count
				"01",                       // output 0: asset version
				"00",
				"34", // output 0: serialization length
				"a69849e11add96ac7053aad22ba2349a4abf5feb0475a0afcadff4e128be76cf", // output 0: assetID
				"92c30f",             // output 0: amount
				"01",                 // output 0: version
				"04",                 // output 0: control program length
				"74727565",           // output 0: control program
				"0109",               // output 0: state data length
				"737461746544617461", // output 0: state data
				"00",                 // output 0: witness length
			}, ""),
			hash: testutil.MustDecodeHash("509632e139a418be372cd4b73c8de7b8aa5215c47848bf76ed2488d0a5c0c55e"),
		},
		{
			tx: NewTx(TxData{
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
			hex: strings.Join([]string{
				"07",                 // serflags
				"01",                 // transaction version
				"00",                 // tx time range
				"01",                 // inputs count
				"01",                 // input 0: asset version
				"0b",                 // input 0: input commitment length
				"02",                 // input 0: coinbase type flag
				"09",                 // input 0: arbitrary length
				"617262697472617279", // input 0: arbitrary data
				"00",                 // input 0: witness length
				"02",                 // outputs count
				"01",                 // output 0: asset version
				"00",
				"34", // output 0: serialization length
				"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", // output 0: assetID
				"92c30f",             // output 0: amount
				"01",                 // output 0: version
				"04",                 // output 0: control program length
				"74727565",           // output 0: control program
				"0109",               // output 0: state data length
				"737461746544617461", // output 0: state data
				"00",                 // output 0: witness length
				"01",                 // output 1: asset version
				"00",
				"35", // output 1: serialization length
				"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", // output 1: assetID
				"92c30f",             // output 1: amount
				"01",                 // output 1: version
				"05",                 // output 1: control program length
				"66616c7365",         // output 1: control program
				"0109",               // output 0: state data length
				"737461746544617461", // output 0: state data
				"00",                 // output 1: witness length
			}, ""),
			hash: testutil.MustDecodeHash("4243fc12f6e8704c5e4318a2addc648d29fd1c03a8fe1246badafa5efd8b5339"),
		},
	}
	for i, test := range cases {
		got := testutil.Serialize(t, test.tx)
		want, err := hex.DecodeString(test.hex)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(got, want) {
			t.Errorf("test %d: bytes = %x want %x", i, got, want)
		}
		if test.tx.ID != test.hash {
			t.Errorf("test %d: hash = %x want %x", i, test.tx.ID.Bytes(), test.hash.Bytes())
		}

		txJSON, err := json.Marshal(test.tx)
		if err != nil {
			t.Errorf("test %d: error marshaling tx to json: %s", i, err)
		}
		txFromJSON := Tx{}
		if err := json.Unmarshal(txJSON, &txFromJSON); err != nil {
			t.Errorf("test %d: error unmarshaling tx from json: %s", i, err)
		}
		if !testutil.DeepEqual(test.tx.TxData, txFromJSON.TxData) {
			t.Errorf("test %d: types.TxData -> json -> types.TxData: got:\n%s\nwant:\n%s", i, spew.Sdump(txFromJSON.TxData), spew.Sdump(test.tx.TxData))
		}

		tx1 := new(TxData)
		if err := tx1.UnmarshalText([]byte(test.hex)); err != nil {
			t.Errorf("test %d: unexpected err %v", i, err)
		}
		if !testutil.DeepEqual(*tx1, test.tx.TxData) {
			t.Errorf("test %d: tx1 is:\n%swant:\n%s", i, spew.Sdump(*tx1), spew.Sdump(test.tx.TxData))
		}
	}
}

func TestTransactionTrailingGarbage(t *testing.T) {
	// validTxHex is a valid tx, we don't care what's inside as long as it's valid
	validTxHex := `07010001012b00030a0908916133a0d64d1d973b631e226ef95338ad4a536b95635f32f0d04708a6f2a26380a094a58d1d090001010101030102030101002a000000000000000000000000000000000000000000000000000000000000000080a094a58d1d0101010000`
	validTx := Tx{}
	if err := validTx.UnmarshalText([]byte(validTxHex)); err != nil {
		t.Fatal(err)
	}

	invalidTxHex := validTxHex + strings.Repeat("00", 10)
	invalidTx := Tx{}
	if err := invalidTx.UnmarshalText([]byte(invalidTxHex)); err == nil {
		t.Fatal("expected error with trailing garbage but got nil")
	}
}

func TestInvalidIssuance(t *testing.T) {
	hex := strings.Join([]string{
		"07",     // serflags
		"01",     // transaction version
		"00",     // tx maxtime
		"01",     // inputs count
		"01",     // input 0, asset version
		"2b",     // input 0, input commitment length prefix
		"00",     // input 0, input commitment, "issuance" type
		"03",     // input 0, input commitment, nonce length prefix
		"0a0908", // input 0, input commitment, nonce
		"0000000000000000000000000000000000000000000000000000000000000000", // input 0, input commitment, WRONG asset id
		"80a094a58d1d", // input 0, input commitment, amount
		"29",           // input 0, issuance input witness length prefix
		"03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d", // input 0, issuance input witness, initial block
		"00",     // input 0, issuance input witness, asset definition
		"01",     // input 0, issuance input witness, vm version
		"01",     // input 0, issuance input witness, issuance program length prefix
		"01",     // input 0, issuance input witness, issuance program
		"01",     // input 0, issuance input witness, arguments count
		"03",     // input 0, issuance input witness, argument 0 length prefix
		"010203", // input 0, issuance input witness, argument 0
		"01",     // outputs count
		"01",     // output 0, asset version
		"29",     // output 0, output commitment length
		"0000000000000000000000000000000000000000000000000000000000000000", // output 0, output commitment, asset id
		"80a094a58d1d",   // output 0, output commitment, amount
		"01",             // output 0, output commitment, vm version
		"0101",           // output 0, output commitment, control program
		"066f7574707574", // output 0, reference data
		"00",             // output 0, output witness
		"0869737375616e6365",
	}, "")

	tx := new(TxData)
	if err := tx.UnmarshalText([]byte(hex)); errors.Root(err) != errBadAssetID {
		t.Errorf("want errBadAssetID, got %v", err)
	}
}

func TestFuzzUnknownAssetVersion(t *testing.T) {
	rawTx := `07010001012b00030a0908916133a0d64d1d973b631e226ef95338ad4a536b95635f32f0d04708a6f2a26380a094a58d1d090001010101030102030101002a000000000000000000000000000000000000000000000000000000000000000080a094a58d1d0101010000`
	/*
		07
		01
		00
		01
		01
		2b
		00
		03
		0a0908
		916133a0d64d1d973b631e226ef95338ad4a536b95635f32f0d04708a6f2a263
		80a094a58d1d0900
		0101010103010203010100
		29
		0000000000000000000000000000000000000000000000000000000000000000
		80a094a58d1d
		01
		01
		01
		00
	*/
	want := Tx{}
	if err := want.UnmarshalText([]byte(rawTx)); err != nil {
		t.Fatal(err)
	}

	b, err := want.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	got := Tx{}
	if err = got.UnmarshalText(b); err != nil {
		t.Fatal(err)
	}
	if got.ID.String() != want.ID.String() {
		t.Errorf("tx id changed to %s", got.ID.String())
	}
}

func BenchmarkTxWriteToTrue(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, 0)
	}
}

func BenchmarkTxWriteToFalse(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, serRequired)
	}
}

func BenchmarkTxWriteToTrue200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil, nil))
		tx.Outputs = append(tx.Outputs, NewOriginalTxOutput(bc.AssetID{}, 0, nil, nil))
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, 0)
	}
}

func BenchmarkTxWriteToFalse200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil, nil))
		tx.Outputs = append(tx.Outputs, NewOriginalTxOutput(bc.AssetID{}, 0, nil, nil))
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, serRequired)
	}
}

func BenchmarkTxInputWriteToTrue(b *testing.B) {
	input := NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew)
	}
}

func BenchmarkTxInputWriteToFalse(b *testing.B) {
	input := NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew)
	}
}

func BenchmarkTxOutputWriteToTrue(b *testing.B) {
	output := NewOriginalTxOutput(bc.AssetID{}, 0, nil, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew)
	}
}

func BenchmarkTxOutputWriteToFalse(b *testing.B) {
	output := NewOriginalTxOutput(bc.AssetID{}, 0, nil, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew)
	}
}

func BenchmarkAssetAmountWriteTo(b *testing.B) {
	aa := bc.AssetAmount{}
	for i := 0; i < b.N; i++ {
		aa.WriteTo(ioutil.Discard)
	}
}
