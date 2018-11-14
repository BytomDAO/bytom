// Package pseudohsm provides a pseudo HSM for development environments.
package pseudohsm

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pborman/uuid"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/errors"
	mnem "github.com/bytom/wallet/mnemonic"
)

// pre-define errors for supporting bytom errorFormatter
var (
	ErrDuplicateKeyAlias = errors.New("duplicate key alias")
	ErrLoadKey           = errors.New("key not found or wrong password ")
	ErrDecrypt           = errors.New("could not decrypt key with given passphrase")
)

// HSM type for storing pubkey and privatekey
type HSM struct {
	cacheMu  sync.Mutex
	keyStore keyStore
	cache    *keyCache
	//kdCache  map[chainkd.XPub]chainkd.XPrv
}

// XPub type for pubkey for anyone can see
type XPub struct {
	Alias string       `json:"alias"`
	XPub  chainkd.XPub `json:"xpub"`
	File  string       `json:"file"`
}

// New method for HSM struct
func New(keypath string) (*HSM, error) {
	keydir, _ := filepath.Abs(keypath)
	return &HSM{
		keyStore: &keyStorePassphrase{keydir, LightScryptN, LightScryptP},
		cache:    newKeyCache(keydir),
		//kdCache:  make(map[chainkd.XPub]chainkd.XPrv),
	}, nil
}

// XCreate produces a new random xprv and stores it in the db.
func (h *HSM) XCreate(alias string, auth string, language string) (*XPub, *string, error) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()

	normalizedAlias := strings.ToLower(strings.TrimSpace(alias))
	if ok := h.cache.hasAlias(normalizedAlias); ok {
		return nil, nil, ErrDuplicateKeyAlias
	}

	xpub, mnemonic, err := h.createChainKDKey(normalizedAlias, auth, language)
	if err != nil {
		return nil, nil, err
	}
	h.cache.add(*xpub)
	return xpub, mnemonic, err
}

// ImportKeyFromMnemonic produces a xprv from mnemonic and stores it in the db.
func (h *HSM) ImportKeyFromMnemonic(alias string, auth string, mnemonic string, language string) (*XPub, error) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()

	normalizedAlias := strings.ToLower(strings.TrimSpace(alias))
	if ok := h.cache.hasAlias(normalizedAlias); ok {
		return nil, ErrDuplicateKeyAlias
	}

	// Pre validate that the mnemonic is well formed and only contains words that
	// are present in the word list
	if !mnem.IsMnemonicValid(mnemonic, language) {
		return nil, mnem.ErrInvalidMnemonic
	}

	xpub, err := h.createKeyFromMnemonic(alias, auth, mnemonic)
	if err != nil {
		return nil, err
	}

	h.cache.add(*xpub)
	return xpub, nil
}

func (h *HSM) createKeyFromMnemonic(alias string, auth string, mnemonic string) (*XPub, error) {
	// Generate a Bip32 HD wallet for the mnemonic and a user supplied password
	seed := mnem.NewSeed(mnemonic, "")
	xprv, xpub, err := chainkd.NewXKeys(bytes.NewBuffer(seed))
	if err != nil {
		return nil, err
	}
	id := uuid.NewRandom()
	key := &XKey{
		ID:      id,
		KeyType: "bytom_kd",
		XPub:    xpub,
		XPrv:    xprv,
		Alias:   alias,
	}
	file := h.keyStore.JoinPath(keyFileName(key.ID.String()))
	if err := h.keyStore.StoreKey(file, key, auth); err != nil {
		return nil, errors.Wrap(err, "storing keys")
	}
	return &XPub{XPub: xpub, Alias: alias, File: file}, nil
}

func (h *HSM) createChainKDKey(alias string, auth string, language string) (*XPub, *string, error) {
	// Generate a mnemonic for memorization or user-friendly seeds
	entropy, err := mnem.NewEntropy(256)
	if err != nil {
		return nil, nil, err
	}
	mnemonic, err := mnem.NewMnemonic(entropy, language)
	if err != nil {
		return nil, nil, err
	}
	xpub, err := h.createKeyFromMnemonic(alias, auth, mnemonic)
	if err != nil {
		return nil, nil, err
	}
	return xpub, &mnemonic, nil
}

// UpdateKeyAlias update key alias
func (h *HSM) UpdateKeyAlias(xpub chainkd.XPub, newAlias string) error {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()

	h.cache.maybeReload()
	h.cache.mu.Lock()
	xpb, err := h.cache.find(XPub{XPub: xpub})
	h.cache.mu.Unlock()
	if err != nil {
		return err
	}

	keyjson, err := ioutil.ReadFile(xpb.File)
	if err != nil {
		return err
	}

	encrptKeyJSON := new(encryptedKeyJSON)
	if err := json.Unmarshal(keyjson, encrptKeyJSON); err != nil {
		return err
	}

	normalizedAlias := strings.ToLower(strings.TrimSpace(newAlias))
	if ok := h.cache.hasAlias(normalizedAlias); ok {
		return ErrDuplicateKeyAlias
	}

	encrptKeyJSON.Alias = normalizedAlias
	keyJSON, err := json.Marshal(encrptKeyJSON)
	if err != nil {
		return err
	}

	if err := writeKeyFile(xpb.File, keyJSON); err != nil {
		return err
	}

	// update key alias
	h.cache.delete(xpb)
	xpb.Alias = normalizedAlias
	h.cache.add(xpb)

	return nil
}

// ListKeys returns a list of all xpubs from the store
func (h *HSM) ListKeys() []XPub {
	xpubs := h.cache.keys()
	return xpubs
}

// XSign looks up the xprv given the xpub, optionally derives a new
// xprv with the given path (but does not store the new xprv), and
// signs the given msg.
func (h *HSM) XSign(xpub chainkd.XPub, path [][]byte, msg []byte, auth string) ([]byte, error) {
	xprv, err := h.LoadChainKDKey(xpub, auth)
	if err != nil {
		return nil, err
	}
	if len(path) > 0 {
		xprv = xprv.Derive(path)
	}
	return xprv.Sign(msg), nil
}

//LoadChainKDKey get xprv from xpub
func (h *HSM) LoadChainKDKey(xpub chainkd.XPub, auth string) (xprv chainkd.XPrv, err error) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()

	//if xprv, ok := h.kdCache[xpub]; ok {
	//	return xprv, nil
	//}

	_, xkey, err := h.loadDecryptedKey(xpub, auth)
	if err != nil {
		return xprv, ErrLoadKey
	}
	//h.kdCache[xpb.XPub] = xkey.XPrv
	return xkey.XPrv, nil
}

// XDelete deletes the key matched by xpub if the passphrase is correct.
// If a contains no filename, the address must match a unique key.
func (h *HSM) XDelete(xpub chainkd.XPub, auth string) error {
	// Decrypting the key isn't really necessary, but we do
	// it anyway to check the password and zero out the key
	// immediately afterwards.

	xpb, xkey, err := h.loadDecryptedKey(xpub, auth)
	if xkey != nil {
		zeroKey(xkey)
	}
	if err != nil {
		return err
	}

	h.cacheMu.Lock()
	// The order is crucial here. The key is dropped from the
	// cache after the file is gone so that a reload happening in
	// between won't insert it into the cache again.
	err = os.Remove(xpb.File)
	if err == nil {
		h.cache.delete(xpb)
	}
	h.cacheMu.Unlock()
	return err
}

func (h *HSM) loadDecryptedKey(xpub chainkd.XPub, auth string) (XPub, *XKey, error) {
	h.cache.maybeReload()
	h.cache.mu.Lock()
	xpb, err := h.cache.find(XPub{XPub: xpub})

	h.cache.mu.Unlock()
	if err != nil {
		return xpb, nil, err
	}
	xkey, err := h.keyStore.GetKey(xpb.Alias, xpb.File, auth)
	return xpb, xkey, err
}

// ResetPassword reset passphrase for an existing xpub
func (h *HSM) ResetPassword(xpub chainkd.XPub, oldAuth, newAuth string) error {
	xpb, xkey, err := h.loadDecryptedKey(xpub, oldAuth)
	if err != nil {
		return err
	}
	return h.keyStore.StoreKey(xpb.File, xkey, newAuth)
}

// HasAlias check whether the key alias exists
func (h *HSM) HasAlias(alias string) bool {
	return h.cache.hasAlias(alias)
}

// HasKey check whether the private key exists
func (h *HSM) HasKey(xprv chainkd.XPrv) bool {
	return h.cache.hasKey(xprv.XPub())
}
