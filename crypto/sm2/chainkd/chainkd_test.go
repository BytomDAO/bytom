package chainkd

import (
	"testing"
)

func TestSign(t *testing.T) {
	xprv, xpub, err := NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("1234567890")
	sig := xprv.Sign(msg)

	if !xpub.Verify(msg, sig) {
		t.Fatal("Verify Fatal")
	}
}
