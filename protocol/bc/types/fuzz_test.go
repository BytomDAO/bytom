package types

import "testing"

func TestFuzzUnknownAssetVersion(t *testing.T) {
	const rawTx = `0701000001012b00030a0908916133a0d64d1d973b631e226ef95338ad4a536b95635f32f0d04708a6f2a26380a094a58d1d09000101010103010203010129000000000000000000000000000000000000000000000000000000000000000080a094a58d1d01010100`

	var want Tx
	err := want.UnmarshalText([]byte(rawTx))
	if err != nil {
		t.Fatal(err)
	}

	b, err := want.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	// Make sure serialzing and deserializing gives the same tx
	var got Tx
	err = got.UnmarshalText(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID.String() != want.ID.String() {
		t.Errorf("tx id changed to %s", got.ID.String())
	}
}
