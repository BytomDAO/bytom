package pseudohsm

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"bytom/blockchain/config"
	"bytom/crypto/ed25519"
	"bytom/errors"
	"bytom/protocol/bc/legacy"
	"bytom/testutil"
)

const dirPath = "D:\\gotest"

func TestPseudoHSMChainKDKeys(t *testing.T) {
	hsm := New(&config.Config{KeyPath: dirPath})
	xpub, err := hsm.XCreate("password", "")
	if err != nil {
		t.Fatal(err)
	}
	xpub2, err := hsm.XCreate("nopassword", "bytom")
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("In the face of ignorance and resistance I wrote financial systems into existence")
	sig, err := hsm.XSign(xpub.XPub, nil, msg, "")
	if err != nil {
		t.Fatal(err)
	}
	if !xpub.XPub.Verify(msg, sig) {
		t.Error("expected verify to succeed")
	}
	if xpub2.XPub.Verify(msg, sig) {
		t.Error("expected verify with wrong pubkey to fail")
	}
	path := [][]byte{{3, 2, 6, 3, 8, 2, 7}}
	sig, err = hsm.XSign(xpub2.XPub, path, msg, "bytom")
	if err != nil {
		t.Fatal(err)
	}
	if xpub2.XPub.Verify(msg, sig) {
		t.Error("expected verify with underived pubkey of sig from derived privkey to fail")
	}
	if !xpub2.XPub.Derive(path).Verify(msg, sig) {
		t.Error("expected verify with derived pubkey of sig from derived privkey to succeed")
	}
	xpubs, _, err := hsm.ListKeys(nil, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(xpubs) != 2 {
		t.Error("expected 2 entries in the db")
	}
}

func TestKeyWithAlias(t *testing.T) {

	hsm := New(&config.Config{KeyPath: dirPath})
	xpub, err := hsm.XCreate(ctx, "some-alias")
	if err != nil {
		t.Fatal(err)
	}

	// List keys, no alias filter
	xpubs, _, err := hsm.ListKeys(nil, 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(xpubs[0], xpub) {
		t.Fatalf("expected to get %v instead got %v", spew.Sdump(xpub), spew.Sdump(xpubs[0]))
	}

	// List keys, with matching alias filter
	xpubs, _, err = hsm.ListKeys([]string{"some-alias", "other-alias"}, 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(xpubs) != 1 {
		t.Fatalf("list keys with matching filter expected to get 1 instead got %v", len(xpubs))
	}

	if !testutil.DeepEqual(xpubs[0], xpub) {
		t.Fatalf("expected to get %v instead got %v", spew.Sdump(xpub), spew.Sdump(xpubs[0]))
	}

	// List keys, with non-matching alias filter
	xpubs, _, err = hsm.ListKeys([]string{"other-alias"}, 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(xpubs) != 0 {
		t.Fatalf("list keys with matching filter expected to get 0 instead got %v", len(xpubs))
	}

	// check for uniqueness error
	xpub, err = hsm.XCreate("xixi", "some-alias")
	if xpub != nil {
		t.Fatalf("xpub: got %v want nil", xpub)
	}
	if errors.Root(err) != ErrDuplicateKeyAlias {
		t.Fatalf("error return value: got %v want %v", errors.Root(err), ErrDuplicateKeyAlias)
	}
}

func TestKeyWithEmptyAlias(t *testing.T) {
	hsm := New(&config.Config{KeyPath: dirPath})
	for i := 0; i < 2; i++ {
		_, err := hsm.XCreate("xx", "")
		if errors.Root(err) != nil {
			t.Fatal(err)
		}
	}
}

func TestKeyOrdering(t *testing.T) {
	hsm := New(&config.Config{KeyPath: dirPath})
	auth := "nowpasswd"
	xpub1, err := hsm.XCreate(auth, "first-key")
	if err != nil {
		t.Fatal(err)
	}

	xpub2, err := hsm.XCreate(auth, "second-key")
	if err != nil {
		t.Fatal(err)
	}

	xpubs, _, err := hsm.ListKeys(nil, 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	// Latest key is returned first
	if !testutil.DeepEqual(xpubs[0], xpub2) {
		t.Fatalf("expected to get %v instead got %v", spew.Sdump(xpub2), spew.Sdump(xpubs[0]))
	}

	_, after, err := hsm.ListKeys(nil, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Older key is returned in second page
	if !testutil.DeepEqual(xpubs[0], xpub1) {
		t.Fatalf("expected to get %v instead got %v", spew.Sdump(xpub1), spew.Sdump(xpubs[0]))
	}
}

func BenchmarkSign(b *testing.B) {
	b.StopTimer()
	auth := "nowpasswd"

	hsm := New(&config.Config{KeyPath: dirPath})
	xpub, err := hsm.XCreate(auth, "")
	if err != nil {
		b.Fatal(err)
	}

	msg := []byte("In the face of ignorance and resistance I wrote financial systems into existence")

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := hsm.XSign(xpub.XPub, nil, msg, auth)
		if err != nil {
			b.Fatal(err)
		}
	}
}
