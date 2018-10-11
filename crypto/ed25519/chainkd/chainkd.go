package chainkd

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"io"

	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/crypto/ed25519/ecmath"
)

type (
	//XPrv external private key
	XPrv [64]byte
	//XPub external public key
	XPub [64]byte
)

// NewXPrv takes a source of random bytes and produces a new XPrv.
// If r is nil, crypto/rand.Reader is used.
func NewXPrv(r io.Reader) (xprv XPrv, err error) {
	if r == nil {
		r = rand.Reader
	}
	var entropy [64]byte
	_, err = io.ReadFull(r, entropy[:])
	if err != nil {
		return xprv, err
	}
	return RootXPrv(entropy[:]), nil
}

// RootXPrv takes a seed binary string and produces a new xprv.
func RootXPrv(seed []byte) (xprv XPrv) {
	h := hmac.New(sha512.New, []byte{'R', 'o', 'o', 't'})
	h.Write(seed)
	h.Sum(xprv[:0])
	pruneRootScalar(xprv[:32])
	return
}

// XPub derives an extended public key from a given xprv.
func (xprv XPrv) XPub() (xpub XPub) {
	var scalar ecmath.Scalar
	copy(scalar[:], xprv[:32])

	var P ecmath.Point
	P.ScMulBase(&scalar)
	buf := P.Encode()

	copy(xpub[:32], buf[:])
	copy(xpub[32:], xprv[32:])

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
	pruneRootScalar(res[:32])
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

// Child derives a child xpub based on `selector` string.
// The corresponding child xprv can be derived from the parent xprv
// using non-hardened derivation: `parentxprv.Child(sel, false)`.
func (xpub XPub) Child(sel []byte) (res XPub) {
	h := hmac.New(sha512.New, xpub[32:])
	h.Write([]byte{'N'})
	h.Write(xpub[:32])
	h.Write(sel)
	h.Sum(res[:0])

	pruneIntermediateScalar(res[:32])

	var (
		f ecmath.Scalar
		F ecmath.Point
	)
	copy(f[:], res[:32])
	F.ScMulBase(&f)

	var (
		pubkey [32]byte
		P      ecmath.Point
	)
	copy(pubkey[:], xpub[:32])
	_, ok := P.Decode(pubkey)
	if !ok {
		panic("XPub should have been validated on initialization")
	}

	P.Add(&P, &F)
	pubkey = P.Encode()
	copy(res[:32], pubkey[:])

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

// Sign creates an EdDSA signature using expanded private key
// derived from the xprv.
func (xprv XPrv) Sign(msg []byte) []byte {
	return Ed25519InnerSign(xprv.ExpandedPrivateKey(), msg)
}

// Verify checks an EdDSA signature using public key
// extracted from the first 32 bytes of the xpub.
func (xpub XPub) Verify(msg []byte, sig []byte) bool {
	return ed25519.Verify(xpub.PublicKey(), msg, sig)
}

// ExpandedPrivateKey generates a 64-byte key where
// the first half is the scalar copied from xprv,
// and the second half is the `prefix` is generated via PRF
// from the xprv.
func (xprv XPrv) ExpandedPrivateKey() ExpandedPrivateKey {
	var res [64]byte
	h := hmac.New(sha512.New, []byte{'E', 'x', 'p', 'a', 'n', 'd'})
	h.Write(xprv[:])
	h.Sum(res[:0])
	copy(res[:32], xprv[:32])
	return res[:]
}

// PublicKey extracts the ed25519 public key from an xpub.
func (xpub XPub) PublicKey() ed25519.PublicKey {
	return ed25519.PublicKey(xpub[:32])
}

// s must be >= 32 bytes long and gets rewritten in place.
// This is NOT the same pruning as in Ed25519: it additionally clears the third
// highest bit to ensure subkeys do not overflow the second highest bit.
func pruneRootScalar(s []byte) {
	s[0] &= 248
	s[31] &= 31 // clear top 3 bits
	s[31] |= 64 // set second highest bit
}

// Clears lowest 3 bits and highest 23 bits of `f`.
func pruneIntermediateScalar(f []byte) {
	f[0] &= 248 // clear bottom 3 bits
	f[29] &= 1  // clear 7 high bits
	f[30] = 0   // clear 8 bits
	f[31] = 0   // clear 8 bits
}
