package pseudohsm

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/pborman/uuid"
)

const (
	veryLightScryptN = 2
	veryLightScryptP = 1
)

// Tests that a json key file can be decrypted and encrypted in multiple rounds.
func TestKeyEncryptDecrypt(t *testing.T) {
	keyjson, err := ioutil.ReadFile("testdata/bytom-very-light-scrypt.json")
	if err != nil {
		t.Fatal(err)
	}
	password := "bytomtest"
	alias := "verylight"
	// Do a few rounds of decryption and encryption
	for i := 0; i < 3; i++ {
		// Try a bad password first

		if _, err := DecryptKey(keyjson, password+"bad"); err == nil {
			t.Errorf("test %d: json key decrypted with bad password", i)
		}

		// Decrypt with the correct password
		key, err := DecryptKey(keyjson, password)
		if err != nil {
			t.Errorf("test %d: json key failed to decrypt: %v", i, err)
		}
		if key.Alias != alias {
			t.Errorf("test %d: key address mismatch: have %x, want %x", i, key.Alias, alias)
		}

		// Recrypt with a new password and start over
		//password += "new data appended"
		if _, err = EncryptKey(key, password, veryLightScryptN, veryLightScryptP); err != nil {
			t.Errorf("test %d: failed to recrypt key %v", i, err)
		}
	}
}

func TestGenerateFile(t *testing.T) {
	xprv, xpub, err := chainkd.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewRandom()
	key := &XKey{
		ID:      id,
		KeyType: "bytom_kd",
		XPub:    xpub,
		XPrv:    xprv,
		Alias:   "verylight",
	}
	t.Log(key)
	password := "bytomtest"
	xkey, err := EncryptKey(key, password, veryLightScryptN, veryLightScryptP)
	file := keyFileName(key.ID.String())
	writeKeyFile(file, xkey)
	os.Remove(file)
}
