package bc

import (
	"testing"
	"encoding/hex"

	"golang.org/x/crypto/sha3"
)

func TestComputeAssetID(t *testing.T) {
	issuanceScript := []byte{1}
	assetID := ComputeAssetID(issuanceScript, 1, &EmptyStringHash)

	unhashed := append([]byte{})
	unhashed = append(unhashed, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...) // vmVersion
	unhashed = append(unhashed, 0x01)                                                      // length of issuanceScript
	unhashed = append(unhashed, issuanceScript...)
	unhashed = append(unhashed, EmptyStringHash.Bytes()...)

	if want := NewAssetID(sha3.Sum256(unhashed)); assetID != want {
		t.Errorf("asset id = %x want %x", assetID.Bytes(), want.Bytes())
	}
}

func TestComputeAssetIDReally(t *testing.T) {
	cases := []struct {
		program       string
		rawDefinition string
		wantAssetID   string
	}{
		{
			program: "ae2039294f652632eee970765550c245f0b0314256b4b93aadc86279fdb45db3b70e5151ad",
			rawDefinition: "7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d",
			wantAssetID: "07c7ced3f37f48ea39da6971c89f90e9cff3202d54b0a911f12ace8501f3834e",
		},
		{
			program: "ae20620b1755451738b04f42822f4b37186563f824c9c30d485987298918f96395fe5151ad",
			rawDefinition: "7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d6f626f6c223a2022220a7d",
			wantAssetID: "0dafd0f0e42f06f3bf9a8cf5787519d3860650f27a2b3393d34e1fe06e89b469",
		},
		{
			program: "ae20db11f9dfa39c9e66421c530fe027218edd3d5b1cd98f24c826f4d9c0cd131a475151ad",
			rawDefinition: "7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d",
			wantAssetID: "a5bc30d8d0ad051e6e352ebc21d79ba798cd8c436e89f4149969c2c562371791",
		},
	}

	for _, c := range cases {
		progBytes, err := hex.DecodeString(c.program)
		if err != nil {
			t.Fatal(err)
		}

		defBytes, err := hex.DecodeString(c.rawDefinition)
		if err != nil {
			t.Fatal(err)
		}

		defHash := NewHash(sha3.Sum256(defBytes))
		assetID := ComputeAssetID(progBytes, 1, &defHash)
		if assetID.String() != c.wantAssetID {
			t.Errorf("got asset id:%s, want asset id:%s", assetID.String(), c.wantAssetID)
		}
	}
}

var assetIDSink AssetID

func BenchmarkComputeAssetID(b *testing.B) {
	var (
		issuanceScript = []byte{5}
	)

	for i := 0; i < b.N; i++ {
		assetIDSink = ComputeAssetID(issuanceScript, 1, &EmptyStringHash)
	}
}
