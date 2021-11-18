package pseudohsm

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/bytom/bytom/crypto/sm2/chainkd"
)

func TestRestore(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	hsm, err := New(dirPath)
	if err != nil {
		t.Fatal("create hsm err:", err)
	}

	testCases := []struct {
		image    string
		wantErr  error
		wantKeys []string
	}{
		{
			image:    "{\"xkeys\":[{\"crypto\": {\"cipher\": \"sm4-128-ctr\",\"ciphertext\": \"2c9a91f1278637e01c31fc52ce1dd00e0a1bdbfe194fefe921cdeade73d0d134bae0a3169739ff82e4f1dd32c13dc03663f9a7c457acb836dc4361a1f05bc0a4\",\"cipherparams\": {\"iv\": \"d4249757a19665095cf0bd0aeac40c06\"}, \"kdf\": \"scrypt\",\"kdfparams\": {\"dklen\": 32,\"n\": 4096,\"p\": 6,\"r\": 8,\"salt\": \"1cfae1c660f6b61ac8a928379bbaa5692793d9cddb4e29a6e9614caf76083112\"},\"mac\": \"1b36ffb1d2c95eff0df836ad7fd75a5995c04ead3cae031fdf7deaf53a7064ff\"},\"id\": \"c538af4c-a8aa-4d4f-b523-9f0e6d286d3f\",\"type\": \"bytom_kd\",\"version\": 1,\"alias\": \"test\",\"xpub\": \"00b5de06f3c513cd7db1ae372844a1d1bb27c15a8e3a7edc2671c9617615f1fb868eab711d2e2e309c979deabb097bffb55227bdfb0f34ab71c844e88fa1aa0cc8\"}]}",
			wantErr:  nil,
			wantKeys: []string{"00b5de06f3c513cd7db1ae372844a1d1bb27c15a8e3a7edc2671c9617615f1fb868eab711d2e2e309c979deabb097bffb55227bdfb0f34ab71c844e88fa1aa0cc8"},
		},
	}

	for _, test := range testCases {
		keyImage := &KeyImage{}
		if err := json.Unmarshal([]byte(test.image), keyImage); err != nil {
			t.Fatal("unmarshal json error:", err)
		}

		if err := hsm.Restore(keyImage); err != test.wantErr {
			t.Errorf("error mismatch: have %v, want %v", err, test.wantErr)
		}

		if len(hsm.cache.keys()) != len(test.wantKeys) {
			t.Errorf("error key num: have %v, want %v", len(hsm.cache.keys()), len(test.wantKeys))
		}

		for _, key := range test.wantKeys {
			var xPub chainkd.XPub
			data, _ := hex.DecodeString(key)
			copy(xPub[:], data)

			if !hsm.cache.hasKey(xPub) {
				t.Errorf("error restore key: can't find key %v", key)
			}
		}
	}
}
