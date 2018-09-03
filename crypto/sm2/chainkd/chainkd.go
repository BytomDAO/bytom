package chainkd

import (
	"crypto/hmac"
	"crypto/sha512"
	"math/big"

	"github.com/bytom/crypto/sm2"
)

type (
	//XPrv external private key
	XPrv [64]byte
	//XPub external public key
	XPub [65]byte
)

// XPub derives an extended public key from a given xprv.
func (xprv XPrv) XPub() (xpub XPub) {
	// var scalar ecmath.Scalar
	// copy(scalar[:], xprv[:32])

	// var P ecmath.Point
	// P.ScMulBase(&scalar)
	// buf := P.Encode()

	// copy(xpub[:32], buf[:])
	// copy(xpub[32:], xprv[32:])

	// return

	privkey := make([]byte, 32)
	copy(privkey, xprv[:32])
	k := new(big.Int).SetBytes(privkey)
	c := sm2.P256Sm2()
	priv := new(sm2.PrivateKey)
	priv.PublicKey.Curve = c
	priv.D = k
	priv.PublicKey.X, priv.PublicKey.Y = c.ScalarBaseMult(k.Bytes())
	compPubkey := sm2.Compress(&priv.PublicKey)

	copy(xpub[:33], compPubkey[:])
	copy(xpub[33:], xprv[32:])

	return
}

// Child derives a child xpub based on `selector` string.
// The corresponding child xprv can be derived from the parent xprv
// using non-hardened derivation: `parentxprv.Child(sel, false)`.
func (xpub XPub) Child(sel []byte) (xpubkey XPub) {
	res := make([]byte, 64)
	h := hmac.New(sha512.New, xpub[33:])
	h.Write([]byte{'N'})
	h.Write(xpub[:33])
	h.Write(sel)
	h.Sum(res[:0])

	k := new(big.Int).SetBytes(res[:32])
	c := sm2.P256Sm2()
	priv := new(sm2.PrivateKey)
	priv.PublicKey.Curve = c
	priv.D = k
	// child pubkey point x and y
	priv.PublicKey.X, priv.PublicKey.Y = c.ScalarBaseMult(k.Bytes())

	parPubkey := sm2.Decompress(xpub[:33])
	newPubkey := new(sm2.PublicKey)
	newPubkey.Curve = c
	newPubkey.X, newPubkey.Y = c.Add(priv.PublicKey.X, priv.PublicKey.Y, parPubkey.X, parPubkey.Y)
	compPubkey := sm2.Compress(newPubkey)

	copy(xpubkey[:33], compPubkey[:])
	copy(xpubkey[33:], res[32:])

	return
}

// Derive generates a child xpub by recursively deriving
// non-hardened child xpubs over the list of selectors:
// `Derive([a,b,c,...]) == Child(a).Child(b).Child(c)...`
func (xpub XPub) Derive(path [][]byte) XPub {
	res := xpub
	for _, p := range path {
		res = res.Child(p)
	}
	return res
}

// PublicKey extracts the sm2 public key from an xpub.
func (xpub XPub) PublicKey() sm2.PubKey {
	return sm2.PubKey(xpub[:33])
}
