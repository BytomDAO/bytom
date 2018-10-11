package chainkd

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"io"
	"math/big"

	"github.com/bytom/crypto/sm2"
)

// Utility functions

// NewXKeys create new xprivite and xpublickey
func NewXKeys(r io.Reader) (xprv XPrv, xpub XPub, err error) {
	if r == nil {
		r = rand.Reader
	}
	var entropy [32]byte
	_, err = io.ReadFull(r, entropy[:])
	if err != nil {
		return xprv, xpub, err
	}
	h := hmac.New(sha512.New, []byte{'R', 'o', 'o', 't'})
	h.Write(entropy[:])
	h.Sum(xprv[:0])

	c := sm2.P256Sm2()
	k := new(big.Int).SetBytes(xprv[:32])
	priv := new(sm2.PrivateKey)
	priv.PublicKey.Curve = c
	priv.D = k
	priv.PublicKey.X, priv.PublicKey.Y = c.ScalarBaseMult(k.Bytes())
	pubkey := priv.Public().(*(sm2.PublicKey))
	compPubkey := sm2.Compress(pubkey)

	copy(xpub[:33], compPubkey[:])
	copy(xpub[33:], xprv[32:])

	return xprv, xpub, nil
}

func XPubKeys(xpubs []XPub) []sm2.PubKey {
	res := make([]sm2.PubKey, 0, len(xpubs))
	for _, xpub := range xpubs {
		res = append(res, xpub.PublicKey())
	}
	return res
}

func DeriveXPubs(xpubs []XPub, path [][]byte) []XPub {
	res := make([]XPub, 0, len(xpubs))
	for _, xpub := range xpubs {
		d := xpub.Derive(path)
		res = append(res, d)
	}
	return res
}
