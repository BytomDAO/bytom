package types

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/testutil"
)

func TestSerializationVetoInput(t *testing.T) {
	assetID := testutil.MustDecodeAsset("81756fdab39a17163b0ce582ee4ee256fb4d1e156c692b997d608a42ecb38d47")
	arguments := [][]byte{
		[]byte("arguments1"),
		[]byte("arguments2"),
	}

	txInput := TxInput{
		AssetVersion: 1,
		TypedInput: &VetoInput{
			Arguments: arguments,
			Vote:      []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"),
			SpendCommitment: SpendCommitment{
				AssetAmount: bc.AssetAmount{
					AssetId: &assetID,
					Amount:  254354,
				},
				SourceID:       testutil.MustDecodeHash("bef8ff450b877df84174ac5c279fc97da0f507ffe7beef7badf116ea9e2ff041"),
				SourcePosition: 3,
				VMVersion:      1,
				ControlProgram: []byte("vetoProgram"),
				StateData:      [][]byte{[]byte("TestStateData")},
			},
		},
	}

	wantHex := strings.Join([]string{
		"01",   // asset version
		"e401", // input commitment length
		"03",   // veto type flag
		"60",   // veto commitment length
		"bef8ff450b877df84174ac5c279fc97da0f507ffe7beef7badf116ea9e2ff041", // source id
		"81756fdab39a17163b0ce582ee4ee256fb4d1e156c692b997d608a42ecb38d47", // assetID
		"92c30f",                     // amount
		"03",                         // source position
		"01",                         // vm version
		"0b",                         // veto program length
		"7665746f50726f6772616d",     // veto program
		"01",                         // state array length
		"0d",                         // state data length
		"54657374537461746544617461", // state data
		"8001",                       //xpub length
		"6166353934303036613430383337643966303238646161626236643538396466306239313338646165666164353638336535323333633236343632373932313732393461386435333265363038363362636631393636323561333566623863656566666133633039363130656239326463666236353561393437663133323639", //voter xpub
		"17",                   // witness length
		"02",                   // argument array length
		"0a",                   // first argument length
		"617267756d656e747331", // first argument data
		"0a",                   // second argument length
		"617267756d656e747332", // second argument data
	}, "")

	// Test convert struct to hex
	var buffer bytes.Buffer
	if err := txInput.writeTo(&buffer); err != nil {
		t.Fatal(err)
	}

	gotHex := hex.EncodeToString(buffer.Bytes())
	if gotHex != wantHex {
		t.Errorf("serialization bytes = %s want %s", gotHex, wantHex)
	}

	// Test convert hex to struct
	var gotTxInput TxInput
	decodeHex, err := hex.DecodeString(wantHex)
	if err != nil {
		t.Fatal(err)
	}

	if err := gotTxInput.readFrom(blockchain.NewReader(decodeHex)); err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(txInput, gotTxInput) {
		t.Errorf("expected marshaled/unmarshaled tx input to be:\n%sgot:\n%s", spew.Sdump(txInput), spew.Sdump(gotTxInput))
	}
}
