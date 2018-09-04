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

	h := hmac.New(sha512.New, xpub[32:])
	h.Write([]byte{'N'})
	h.Write(xpub[:32])
	h.Write(sel)
	h.Sum(res[:0])

	pruneIntermediateScalar(res[:32])

	// Unrolled the following loop:
	// var carry int
	// carry = 0
	// for i := 0; i < 32; i++ {
	//         sum := int(xprv[i]) + int(res[i]) + carry
	//         res[i] = byte(sum & 0xff)
	//         carry = (sum >> 8)
	// }

	sum := int(0)

	sum = int(xprv[0]) + int(res[0]) + (sum >> 8)
	res[0] = byte(sum & 0xff)
	sum = int(xprv[1]) + int(res[1]) + (sum >> 8)
	res[1] = byte(sum & 0xff)
	sum = int(xprv[2]) + int(res[2]) + (sum >> 8)
	res[2] = byte(sum & 0xff)
	sum = int(xprv[3]) + int(res[3]) + (sum >> 8)
	res[3] = byte(sum & 0xff)
	sum = int(xprv[4]) + int(res[4]) + (sum >> 8)
	res[4] = byte(sum & 0xff)
	sum = int(xprv[5]) + int(res[5]) + (sum >> 8)
	res[5] = byte(sum & 0xff)
	sum = int(xprv[6]) + int(res[6]) + (sum >> 8)
	res[6] = byte(sum & 0xff)
	sum = int(xprv[7]) + int(res[7]) + (sum >> 8)
	res[7] = byte(sum & 0xff)
	sum = int(xprv[8]) + int(res[8]) + (sum >> 8)
	res[8] = byte(sum & 0xff)
	sum = int(xprv[9]) + int(res[9]) + (sum >> 8)
	res[9] = byte(sum & 0xff)
	sum = int(xprv[10]) + int(res[10]) + (sum >> 8)
	res[10] = byte(sum & 0xff)
	sum = int(xprv[11]) + int(res[11]) + (sum >> 8)
	res[11] = byte(sum & 0xff)
	sum = int(xprv[12]) + int(res[12]) + (sum >> 8)
	res[12] = byte(sum & 0xff)
	sum = int(xprv[13]) + int(res[13]) + (sum >> 8)
	res[13] = byte(sum & 0xff)
	sum = int(xprv[14]) + int(res[14]) + (sum >> 8)
	res[14] = byte(sum & 0xff)
	sum = int(xprv[15]) + int(res[15]) + (sum >> 8)
	res[15] = byte(sum & 0xff)
	sum = int(xprv[16]) + int(res[16]) + (sum >> 8)
	res[16] = byte(sum & 0xff)
	sum = int(xprv[17]) + int(res[17]) + (sum >> 8)
	res[17] = byte(sum & 0xff)
	sum = int(xprv[18]) + int(res[18]) + (sum >> 8)
	res[18] = byte(sum & 0xff)
	sum = int(xprv[19]) + int(res[19]) + (sum >> 8)
	res[19] = byte(sum & 0xff)
	sum = int(xprv[20]) + int(res[20]) + (sum >> 8)
	res[20] = byte(sum & 0xff)
	sum = int(xprv[21]) + int(res[21]) + (sum >> 8)
	res[21] = byte(sum & 0xff)
	sum = int(xprv[22]) + int(res[22]) + (sum >> 8)
	res[22] = byte(sum & 0xff)
	sum = int(xprv[23]) + int(res[23]) + (sum >> 8)
	res[23] = byte(sum & 0xff)
	sum = int(xprv[24]) + int(res[24]) + (sum >> 8)
	res[24] = byte(sum & 0xff)
	sum = int(xprv[25]) + int(res[25]) + (sum >> 8)
	res[25] = byte(sum & 0xff)
	sum = int(xprv[26]) + int(res[26]) + (sum >> 8)
	res[26] = byte(sum & 0xff)
	sum = int(xprv[27]) + int(res[27]) + (sum >> 8)
	res[27] = byte(sum & 0xff)
	sum = int(xprv[28]) + int(res[28]) + (sum >> 8)
	res[28] = byte(sum & 0xff)
	sum = int(xprv[29]) + int(res[29]) + (sum >> 8)
	res[29] = byte(sum & 0xff)
	sum = int(xprv[30]) + int(res[30]) + (sum >> 8)
	res[30] = byte(sum & 0xff)
	sum = int(xprv[31]) + int(res[31]) + (sum >> 8)
	res[31] = byte(sum & 0xff)

	if (sum >> 8) != 0 {
		panic("sum does not fit in 256-bit int")
	}
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
