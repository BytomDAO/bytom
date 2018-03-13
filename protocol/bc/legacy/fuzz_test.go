package legacy

import "testing"

func TestFuzzUnknownAssetVersion(t *testing.T) {
	const rawTx = `0701000001012b00030a0908a9b2b6c5394888ab5396f583ae484b8459486b14268e2bef1b637440335eb6c180a094a58d1d2903deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d000101010103010203010129000000000000000000000000000000000000000000000000000000000000000080a094a58d1d01010100`

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
