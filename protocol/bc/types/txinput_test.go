package types

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/encoding/blockchain"
	"github.com/bytom/testutil"
)

func TestSerializationIssuance(t *testing.T) {
	arguments := [][]byte{
		[]byte("arguments1"),
		[]byte("arguments2"),
	}
	issuance := NewIssuanceInput([]byte("nonce"), 254354, []byte("issuanceProgram"), arguments, []byte("assetDefinition"))

	wantHex := strings.Join([]string{
		"01",         // asset version
		"2a",         // serialization length
		"00",         // issuance type flag
		"05",         // nonce length
		"6e6f6e6365", // nonce
		"a69849e11add96ac7053aad22ba2349a4abf5feb0475a0afcadff4e128be76cf", // assetID
		"92c30f", // amount
		"38",     // input witness length
		"0f",     // asset definition length
		"6173736574446566696e6974696f6e", // asset definition
		"01", // vm version
		"0f", // issuanceProgram length
		"69737375616e636550726f6772616d", // issuance program
		"02", // argument array length
		"0a", // first argument length
		"617267756d656e747331", // first argument data
		"0a", // second argument length
		"617267756d656e747332", // second argument data
	}, "")

	// Test convert struct to hex
	var buffer bytes.Buffer
	if err := issuance.writeTo(&buffer); err != nil {
		t.Fatal(err)
	}

	gotHex := hex.EncodeToString(buffer.Bytes())
	if gotHex != wantHex {
		t.Errorf("serialization bytes = %s want %s", gotHex, wantHex)
	}

	// Test convert hex to struct
	var gotIssuance TxInput
	decodeHex, err := hex.DecodeString(wantHex)
	if err != nil {
		t.Fatal(err)
	}

	if err := gotIssuance.readFrom(blockchain.NewReader(decodeHex)); err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(*issuance, gotIssuance) {
		t.Errorf("expected marshaled/unmarshaled txinput to be:\n%sgot:\n%s", spew.Sdump(*issuance), spew.Sdump(gotIssuance))
	}
}

func TestSerializationSpend(t *testing.T) {
	arguments := [][]byte{
		[]byte("arguments1"),
		[]byte("arguments2"),
	}
	spend := NewSpendInput(arguments, testutil.MustDecodeHash("fad5195a0c8e3b590b86a3c0a95e7529565888508aecca96e9aeda633002f409"), testutil.MustDecodeAsset("fe9791d71b67ee62515e08723c061b5ccb952a80d804417c8aeedf7f633c524a"), 254354, 3, []byte("spendProgram"))

	wantHex := strings.Join([]string{
		"01", // asset version
		"54", // input commitment length
		"01", // spend type flag
		"52", // spend commitment length
		"fad5195a0c8e3b590b86a3c0a95e7529565888508aecca96e9aeda633002f409", // source id
		"fe9791d71b67ee62515e08723c061b5ccb952a80d804417c8aeedf7f633c524a", // assetID
		"92c30f", // amount
		"03",     // source position
		"01",     // vm version
		"0c",     // spend program length
		"7370656e6450726f6772616d", // spend program
		"17", // witness length
		"02", // argument array length
		"0a", // first argument length
		"617267756d656e747331", // first argument data
		"0a", // second argument length
		"617267756d656e747332", // second argument data
	}, "")

	// Test convert struct to hex
	var buffer bytes.Buffer
	if err := spend.writeTo(&buffer); err != nil {
		t.Fatal(err)
	}

	gotHex := hex.EncodeToString(buffer.Bytes())
	if gotHex != wantHex {
		t.Errorf("serialization bytes = %s want %s", gotHex, wantHex)
	}

	// Test convert hex to struct
	var gotSpend TxInput
	decodeHex, err := hex.DecodeString(wantHex)
	if err != nil {
		t.Fatal(err)
	}

	if err := gotSpend.readFrom(blockchain.NewReader(decodeHex)); err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(*spend, gotSpend) {
		t.Errorf("expected marshaled/unmarshaled txinput to be:\n%sgot:\n%s", spew.Sdump(*spend), spew.Sdump(gotSpend))
	}
}

func TestSerializationCoinbase(t *testing.T) {
	coinbase := NewCoinbaseInput([]byte("arbitrary"))
	wantHex := strings.Join([]string{
		"01",                 // asset version
		"0b",                 // input commitment length
		"02",                 // coinbase type flag
		"09",                 // arbitrary length
		"617262697472617279", // arbitrary data
		"00",                 // witness length
	}, "")

	// Test convert struct to hex
	var buffer bytes.Buffer
	if err := coinbase.writeTo(&buffer); err != nil {
		t.Fatal(err)
	}

	gotHex := hex.EncodeToString(buffer.Bytes())
	if gotHex != wantHex {
		t.Errorf("serialization bytes = %s want %s", gotHex, wantHex)
	}

	// Test convert hex to struct
	var gotCoinbase TxInput
	decodeHex, err := hex.DecodeString(wantHex)
	if err != nil {
		t.Fatal(err)
	}

	if err := gotCoinbase.readFrom(blockchain.NewReader(decodeHex)); err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(*coinbase, gotCoinbase) {
		t.Errorf("expected marshaled/unmarshaled txinput to be:\n%sgot:\n%s", spew.Sdump(*coinbase), spew.Sdump(gotCoinbase))
	}
}
