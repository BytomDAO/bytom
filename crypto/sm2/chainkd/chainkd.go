package chainkd

import (
	"crypto/hmac"
	"crypto/sha512"
	"errors"
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
	privkey := make([]byte, 32)
	copy(privkey, xprv[:32])
	k := new(big.Int).SetBytes(privkey[:32])
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
	if k.Cmp(c.Params().N) >= 0 || k.Sign() == 0 {
		errors.New("Invalid Child")
		return
	}
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

// Child derives a child xprv based on `selector` string and `hardened` flag.
// If `hardened` is false, child xpub can be derived independently
// from the parent xpub without using the parent xprv.
// If `hardened` is true, child key can only be derived from the parent xprv.
func (xprv XPrv) Child(sel []byte, hardened bool) XPrv {
	if hardened {
		return xprv.hardenedChild(sel)
	}
	return xprv.nonhardenedChild(sel)
}

func (xprv XPrv) hardenedChild(sel []byte) (res XPrv) {
	h := hmac.New(sha512.New, xprv[32:])
	h.Write([]byte{'H'})
	h.Write(xprv[:32])
	h.Write(sel)
	h.Sum(res[:0])

	left := new(big.Int).SetBytes(res[:32])
	k := new(big.Int).SetBytes(xprv[:32])
	r := new(big.Int)
	r.Add(left, k)
	c := sm2.P256Sm2()
	N := c.Params().N
	r.Mod(r, N)
	priv := r.Bytes()

	copy(res[:32], priv[:32])

	return
}

func (xprv XPrv) nonhardenedChild(sel []byte) (res XPrv) {
	xpub := xprv.XPub()

	h := hmac.New(sha512.New, xpub[33:])
	h.Write([]byte{'N'})
	h.Write(xpub[:33])
	h.Write(sel)
	h.Sum(res[:0])

	left := new(big.Int).SetBytes(res[:32])
	k := new(big.Int).SetBytes(xprv[:32])
	r := new(big.Int)
	r.Add(left, k)
	N := sm2.P256Sm2().Params().N
	r.Mod(r, N)
	priv := r.Bytes()

	copy(res[:32], priv[:])

	return
}

// Derive generates a child xprv by recursively deriving
// non-hardened child xprvs over the list of selectors:
// `Derive([a,b,c,...]) == Child(a).Child(b).Child(c)...`
func (xprv XPrv) Derive(path [][]byte) XPrv {
	res := xprv
	for _, p := range path {
		res = res.Child(p, false)
	}

	return res
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

// Sign creates an sm2 signature using expanded private key
// derived from the xprv.
func (xprv XPrv) Sign(msg []byte) []byte {
	priv := new(sm2.PrivateKey)
	k := new(big.Int).SetBytes(xprv[:32])
	c := sm2.P256Sm2()
	priv.D = k
	priv.PublicKey.Curve = c
	priv.PublicKey.X, priv.PublicKey.Y = c.ScalarBaseMult(k.Bytes())

	r, s, err := sm2.Sign(priv, msg)
	if err != nil {
		panic(err)
	}
	R := r.Bytes()
	S := s.Bytes()
	sig := make([]byte, 64)
	copy(sig[:32], R[:])
	copy(sig[32:], S[:])

	return sig
}

// Verify checks an sm2 signature using public key
// extracted from the first 33 bytes of the xpub.
func (xpub XPub) Verify(msg []byte, sig []byte) bool {
	// return ed25519.Verify(xpub.PublicKey(), msg, sig)
	return sm2.VerifyCompressedPubkey(xpub.PublicKey(), msg, sig)
}
