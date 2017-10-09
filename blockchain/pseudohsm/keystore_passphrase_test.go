// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-etherem library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package pseudohsm

import (
	"io/ioutil"
	"testing"

	"github.com/bytom/common"
	"github.com/bytom/crypto"
	"github.com/bytom/crypto/ed25519/chainkd"

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
	address := common.StringToAddress("bm1pcwfm9xnkrf62pg405tcgjzzk7ur670jqhtm3cq")

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
		if key.Address != address {
			t.Errorf("test %d: key address mismatch: have %x, want %x", i, key.Address, address)
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
		Id:      id,
		KeyType: "bytom_kd",
		Address: crypto.PubkeyToAddress(xpub[:]),
		XPub:    xpub,
		XPrv:    xprv,
	}
	t.Log(key)
	password := "bytomtest"
	xkey, err := EncryptKey(key, password, veryLightScryptN, veryLightScryptP)
	writeKeyFile(keyFileName(key.Address), xkey)
	//writeKeyFile("zzz", xkey)
}
