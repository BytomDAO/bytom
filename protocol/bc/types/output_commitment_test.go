package types

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/encoding/blockchain"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
)

func TestSerializationOutputCommitment(t *testing.T) {
	assetID := testutil.MustDecodeAsset("81756fdab39a17163b0ce582ee4ee256fb4d1e156c692b997d608a42ecb38d47")
	oc := &OutputCommitment{
		AssetAmount: bc.AssetAmount{
			AssetId: &assetID,
			Amount:  254354,
		},
		VMVersion:      1,
		ControlProgram: []byte("TestSerializationOutputCommitment"),
	}

	wantHex := strings.Join([]string{
		"46", // serialization length
		"81756fdab39a17163b0ce582ee4ee256fb4d1e156c692b997d608a42ecb38d47", // assetID
		"92c30f", // amount
		"01",     // version
		"21",     // control program length
		"5465737453657269616c697a6174696f6e4f7574707574436f6d6d69746d656e74", // control program
	}, "")

	// Test convert struct to hex
	var buffer bytes.Buffer
	suffix := []byte{}
	if err := oc.writeExtensibleString(&buffer, suffix, 1); err != nil {
		t.Fatal(err)
	} else if len(suffix) != 0 {
		t.Errorf("output commitment write to got garbage hex left")
	}

	gotHex := hex.EncodeToString(buffer.Bytes())
	if gotHex != wantHex {
		t.Errorf("serialization bytes = %s want %s", gotHex, wantHex)
	}

	// Test convert hex to struct
	var gotOC OutputCommitment
	decodeHex, err := hex.DecodeString(wantHex)
	if err != nil {
		t.Fatal(err)
	}

	if getSuffix, err := gotOC.readFrom(blockchain.NewReader(decodeHex), 1); err != nil {
		t.Fatal(err)
	} else if len(getSuffix) != 0 {
		t.Errorf("output commitment read from got garbage hex left")
	}

	if !testutil.DeepEqual(*oc, gotOC) {
		t.Errorf("expected marshaled/unmarshaled output commitment to be:\n%sgot:\n%s", spew.Sdump(*oc), spew.Sdump(gotOC))
	}
}
