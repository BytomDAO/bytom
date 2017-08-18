// Package pseudohsm provides a pseudo HSM for development environments.
package pseudohsm

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"bytom/crypto/ed25519"
	"bytom/crypto/ed25519/chainkd"
	"bytom/errors"
	"bytom/protocol/bc/legacy"
)

// listKeyMaxAliases limits the alias filter to a sane maximum size.
const listKeyMaxAliases = 200

var (
	ErrInvalidAfter         = errors.New("invalid after")
	ErrNoKey                = errors.New("key not found")
	ErrInvalidKeySize       = errors.New("key invalid size")
	ErrTooManyAliasesToList = errors.New("requested aliases exceeds limit")
	ErrAmbiguousAddr		= errors.New("multiple keys match address")
)

type HSM struct {
	cacheMu  sync.Mutex
	keyStore keyStore
	cache    *addrCache
	kdCache  map[chainkd.XPub]chainkd.XPrv
}

type XPub struct {
	Alias    string		   `json:"alias"`
	Address common.Address `json:"address"`
	XPub  chainkd.XPub 	   `json:"xpub"`
	File    string		   `json:"file"`
}


func New(conf *Config.config) *HSM {
	keydir, _ = filepath.Abs(conf.KeyPath)
	return &HSM{
		keyStore:   &keyStorePassphrase{keydir, LightScryptN, LightScryptP}
		addrCache:	newAddrCache(keydir)
		kdCache:	make(map[chainkd.XPub]chainkd.XPrv),
	}
}

// XCreate produces a new random xprv and stores it in the db.
func (h *HSM) XCreate(auth string, alias string) (*XPub, error) {
	xpub, _, err := h.createChainKDKey(auth, alias, false)
	if err != nil {
		h.cache.add(xpub)
	}
	return xpub, err
}

func (h *HSM) createChainKDKey(auth string, alias string, get bool) (*XPub, bool, error) {
	xprv, xpub, err := chainkd.NewXKeys(nil)
	if err != nil {
		return nil, false, err
	}
	id := uuid.NewRandom()
	key := &XKey{
		Id: id, 
		KeyType: "bytom_kd", 
		Address: crypto.PubkeyToAddress(xpub),
		XPub: xpub, 
		XPrv: xprv
	}
	file := h.keyStore.JoinPath(keyFileName(key.Address))
	if err := h.keyStore.StoreKey(file, key, auth); err != nil {
		return nil, false, errors.Wrap(err, "storing keys")
	}
	return &XPub{XPub: xpub, Address: key.Address, Alias: alias, File: file}, true, nil
}


// ListKeys returns a list of all xpubs from the db.
func (h *HSM) ListKeys(aliases []string, after int, limit int) ([]*XPub,  error) {
	if len(aliases) > listKeyMaxAliases {
		return nil, "", errors.WithDetailf(ErrTooManyAliasesToList, "max: %d", listKeyMaxAliases)
	}
	xpubs = h.cache.keys()
	return xpubs[after:after+limit], nil
}

// XSign looks up the xprv given the xpub, optionally derives a new
// xprv with the given path (but does not store the new xprv), and
// signs the given msg.
func (h *HSM) XSign(xpub chainkd.XPub, path [][]byte, msg []byte, auth string) ([]byte, error) {
	xprv, err := h.loadChainKDKey(xpub, auth)
	if err != nil {
		return nil, err
	}
	if len(path) > 0 {
		xprv = xprv.Derive(path)
	}
	return xprv.Sign(msg), nil
}

func (h *HSM) loadChainKDKey(xpub chainkd.XPub, auth string) (xprv chainkd.XPrv, err error) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()

	if xprv, ok := h.kdCache[xpub]; ok {
		return xprv, nil
	}

	key, file, err := h.loadDecryptedKey(xpub, auth)
	if err != nil {
		return nil, ErrNoKey
	}
	h.kdCache[xpub] = key.xprv
	return key.xprv, nil
}


// XDelete deletes the key matched by xpub if the passphrase is correct.
// If a contains no filename, the address must match a unique key.
func (h *HSM) XDelete(xpub chainkd.XPub, auth string) error {
	// Decrypting the key isn't really necessary, but we do
	// it anyway to check the password and zero out the key
	// immediately afterwards.
	
	key, file, err := h.loadDecryptedKey(xpub, auth)
	if key != nil {
		zeroKey(key.XPrv)
	}
	if err != nil {
		return err
	}

	file := h.keyStore.JoinPath(file)

	// The order is crucial here. The key is dropped from the
	// cache after the file is gone so that a reload happening in
	// between won't insert it into the cache again.
	err = os.Remove(file)
	if err == nil {
		h.cache.delete(xpub)
	}
	h.cacheMu.Lock()
	delete(h.kdCache, xpub)
	h.cacheMu.Unlock()
	return err
}

func (h *HSM) loadDecryptedKey(xpub chainkd.XPub, auth string) (*XKey, string, error) {
	h.cache.maybeReload()
	h.cache.mu.Lock()
	key, err := h.cache.find(xpub)
	h.cache.mu.Unlock()
	if err != nil {
		return a, nil, err
	}
	xkey, err := h.keyStore.GetKey(key.Address, key.File, auth)
	return xkey, key.File,  err
}

