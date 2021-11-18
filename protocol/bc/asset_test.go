package bc

import (
	"encoding/hex"
	"testing"

	"github.com/bytom/bytom/crypto/sm3"
)

func TestComputeAssetID(t *testing.T) {
	issuanceScript := []byte{1}
	assetID := ComputeAssetID(issuanceScript, 1, &EmptyStringHash)

	unhashed := append([]byte{})
	unhashed = append(unhashed, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...) // vmVersion
	unhashed = append(unhashed, 0x01)                                                      // length of issuanceScript
	unhashed = append(unhashed, issuanceScript...)
	unhashed = append(unhashed, EmptyStringHash.Bytes()...)

	if want := NewAssetID(sm3.Sum256(unhashed)); assetID != want {
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
			program:       "ae2039294f652632eee970765550c245f0b0314256b4b93aadc86279fdb45db3b70e5151ad",
			rawDefinition: "7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d",
			wantAssetID:   "3f5dc9fa2b7f409d78ca962c7460610d06095c9e747f446092e464ab9d8d6222",
		},
		{
			program:       "ae20620b1755451738b04f42822f4b37186563f824c9c30d485987298918f96395fe5151ad",
			rawDefinition: "7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d6f626f6c223a2022220a7d",
			wantAssetID:   "c295bed672f1e5519fc8cf9885e80b81c8e547544a16d60a94e1b9782e1f0a3f",
		},
		{
			program:       "ae20db11f9dfa39c9e66421c530fe027218edd3d5b1cd98f24c826f4d9c0cd131a475151ad",
			rawDefinition: "7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d",
			wantAssetID:   "c4b377488711107a133f9c43d1e41af8b9ef6a840d61c728add8c5d9530cd644",
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

		defHash := NewHash(sm3.Sum256(defBytes))
		assetID := ComputeAssetID(progBytes, 1, &defHash)
		if assetID.String() != c.wantAssetID {
			t.Errorf("got asset id:%s, want asset id:%s", assetID.String(), c.wantAssetID)
		}
	}
}
