package types

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/testutil"
)

func TestReadWriteOutputCommitment(t *testing.T) {
	btmAssetID := testutil.MustDecodeAsset("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	cases := []struct {
		oc           *OutputCommitment
		encodeString string
	}{
		{
			oc: &OutputCommitment{
				AssetAmount:    bc.AssetAmount{AssetId: &btmAssetID, Amount: 100},
				VMVersion:      1,
				ControlProgram: testutil.MustDecodeHexString("00140876db6ca8f4542a836f0edd42b87d095d081182"),
				StateData:      []byte("stateData1"),
			},
			encodeString: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff64011600140876db6ca8f4542a836f0edd42b87d095d0811820a73746174654461746131",
		},
		{
			oc: &OutputCommitment{
				AssetAmount:    bc.AssetAmount{AssetId: &btmAssetID, Amount: 50},
				VMVersion:      1,
				ControlProgram: testutil.MustDecodeHexString("00148bf7800b2333afd8414d6e903d58c4908b9bbcc7"),
				StateData:      []byte("stateData2"),
			},
			encodeString: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff32011600148bf7800b2333afd8414d6e903d58c4908b9bbcc70a73746174654461746132",
		},
	}

	for _, c := range cases {
		buff := []byte{}
		buffer := bytes.NewBuffer(buff)
		if err := c.oc.writeTo(buffer, 1); err != nil {
			t.Fatal(err)
		}

		got := hex.EncodeToString(buffer.Bytes())
		if got != c.encodeString {
			t.Errorf("got:%s, want:%s", got, c.encodeString)
		}

		oc := &OutputCommitment{}
		if err := oc.readFrom(blockchain.NewReader(buffer.Bytes()), 1); err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(*oc, *c.oc) {
			t.Errorf("got:%v, want:%v", *oc, *c.oc)
		}
	}
}
