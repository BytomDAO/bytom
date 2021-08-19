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

func TestSerializationSpendCommitment(t *testing.T) {
	assetID := testutil.MustDecodeAsset("81756fdab39a17163b0ce582ee4ee256fb4d1e156c692b997d608a42ecb38d47")
	sc := &SpendCommitment{
		AssetAmount: bc.AssetAmount{
			AssetId: &assetID,
			Amount:  254354,
		},
		SourceID:       testutil.MustDecodeHash("bef8ff450b877df84174ac5c279fc97da0f507ffe7beef7badf116ea9e2ff041"),
		SourcePosition: 3,
		VMVersion:      1,
		ControlProgram: []byte("TestSerializationSpendCommitment"),
		StateData:      [][]byte{[]byte("TestStateData")},
	}

	wantHex := strings.Join([]string{
		"75", // serialization length
		"bef8ff450b877df84174ac5c279fc97da0f507ffe7beef7badf116ea9e2ff041", // sourceID
		"81756fdab39a17163b0ce582ee4ee256fb4d1e156c692b997d608a42ecb38d47", // assetID
		"92c30f", // amount
		"03",     // position
		"01",     // version
		"20",     // control program length
		"5465737453657269616c697a6174696f6e5370656e64436f6d6d69746d656e74", // control program
		"010d",                       // stata data length
		"54657374537461746544617461", // state data
	}, "")

	// Test convert struct to hex
	var buffer bytes.Buffer
	suffix := []byte{}
	if err := sc.writeExtensibleString(&buffer, suffix, 1); err != nil {
		t.Fatal(err)
	} else if len(suffix) != 0 {
		t.Errorf("spend commitment write to got garbage hex left")
	}

	gotHex := hex.EncodeToString(buffer.Bytes())
	if gotHex != wantHex {
		t.Errorf("serialization bytes = %s want %s", gotHex, wantHex)
	}

	// Test convert hex to struct
	var gotSC SpendCommitment
	decodeHex, err := hex.DecodeString(wantHex)
	if err != nil {
		t.Fatal(err)
	}

	if getSuffix, err := gotSC.readFrom(blockchain.NewReader(decodeHex), 1); err != nil {
		t.Fatal(err)
	} else if len(getSuffix) != 0 {
		t.Errorf("spend commitment read from got garbage hex left")
	}

	if !testutil.DeepEqual(*sc, gotSC) {
		t.Errorf("expected marshaled/unmarshaled spend commitment to be:\n%sgot:\n%s", spew.Sdump(*sc), spew.Sdump(gotSC))
	}
}
