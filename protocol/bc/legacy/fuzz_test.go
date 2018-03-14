package legacy

import "testing"

func TestFuzzUnknownAssetVersion(t *testing.T) {
	const rawTx = `0701000001012b00030a0908fa48ca4e0150f83fbf26cf83211d136313cde98601a667d999ab9cc27b723d4680a094a58d1d2903deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d000101010103010203010129000000000000000000000000000000000000000000000000000000000000000080a094a58d1d01010100`

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
