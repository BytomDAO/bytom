package bc

import (
	"testing"

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
	want := NewAssetID(sha3.Sum256(unhashed))

	if assetID != want {
		t.Errorf("asset id = %x want %x", assetID.Bytes(), want.Bytes())
	}
}

var assetIDSink AssetID

func BenchmarkComputeAssetID(b *testing.B) {
	var (
		issuanceScript   = []byte{5}
	)

	for i := 0; i < b.N; i++ {
		assetIDSink = ComputeAssetID(issuanceScript, 1, &EmptyStringHash)
	}
}

func mustDecodeHash(s string) (h Hash) {
	err := h.UnmarshalText([]byte(s))
	if err != nil {
		panic(err)
	}
	return h
}
