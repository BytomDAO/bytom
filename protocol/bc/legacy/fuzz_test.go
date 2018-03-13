package legacy

import "testing"

func TestFuzzUnknownAssetVersion(t *testing.T) {
	const rawTx = `07010000000101270eac870dfde1e0feaa4fac6693dee38da2afe7f5cc83ce2b024f04a2400fd6e20a0104deadbeef027b7d00`

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
