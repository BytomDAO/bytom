package pseudohsm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/pborman/uuid"
)

const (
	version = 1
	keytype = "bytom_kd"
)

// XKey struct type for keystore file
type XKey struct {
	ID      uuid.UUID
	KeyType string
	Alias   string
	XPrv    chainkd.XPrv
	XPub    chainkd.XPub
}

type keyStore interface {
	// Loads and decrypts the key from disk.
	GetKey(alias string, filename string, auth string) (*XKey, error)
	// Writes and encrypts the key.
	StoreKey(filename string, k *XKey, auth string) error
	// Joins filename with the key directory unless it is already absolute.
	JoinPath(filename string) string
}

type encryptedKeyJSON struct {
	Crypto  cryptoJSON `json:"crypto"`
	ID      string     `json:"id"`
	Type    string     `json:"type"`
	Version int        `json:"version"`
	Alias   string     `json:"alias"`
	XPub    string     `json:"xpub"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

type scryptParamsJSON struct {
	N     int    `json:"n"`
	R     int    `json:"r"`
	P     int    `json:"p"`
	DkLen int    `json:"dklen"`
	Salt  string `json:"salt"`
}

func writeKeyFile(file string, content []byte) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}

func zeroKey(k *XKey) {
	b := k.XPrv
	for i := range b {
		b[i] = 0
	}
}

// keyFileName implements the naming convention for keyfiles:
// UTC--<created_at UTC ISO8601>-<address hex>
func keyFileName(keyId string) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), keyId)
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}
