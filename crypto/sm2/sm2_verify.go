package sm2

import (
	"math/big"
)

// VerifyBytes verify sigature is valid.
// The parameters is bytes.
func VerifyBytes(pubX, pubY, msg, uid, r, s []byte) bool {
	pub := &PublicKey{
		Curve: P256Sm2(),
		X:     new(big.Int).SetBytes(pubX),
		Y:     new(big.Int).SetBytes(pubY),
	}
	bigR := new(big.Int).SetBytes(r)
	bigS := new(big.Int).SetBytes(s)

	c := P256Sm2()
	N := c.Params().N
	one := new(big.Int).SetInt64(1)
	if bigR.Cmp(one) < 0 || bigS.Cmp(one) < 0 {
		return false
	}
	if bigR.Cmp(N) >= 0 || bigS.Cmp(N) >= 0 {
		return false
	}
	za, err := ZA(pub, uid)
	if err != nil {
		return false
	}
	e, err := msgHash(za, msg)
	if err != nil {
		return false
	}
	t := new(big.Int).Add(bigR, bigS)
	t.Mod(t, N)
	if t.Sign() == 0 {
		return false
	}
	var x *big.Int
	x1, y1 := c.ScalarBaseMult(bigS.Bytes())
	x2, y2 := c.ScalarMult(pub.X, pub.Y, t.Bytes())
	x, _ = c.Add(x1, y1, x2, y2)

	x.Add(x, e)
	x.Mod(x, N)
	return x.Cmp(bigR) == 0
}

// Sm2VerifyBytes verify sigature is valid.
// The parameters is bytes.
// PublicKey length is 64 bytes, it is uncompressed.
func Sm2VerifyBytes(publicKey, hash, signature []byte) bool {
	pub := &PublicKey{
		Curve: P256Sm2(),
		X:     new(big.Int).SetBytes(publicKey[1:33]),
		Y:     new(big.Int).SetBytes(publicKey[33:]),
	}
	bigR := new(big.Int).SetBytes(signature[:32])
	bigS := new(big.Int).SetBytes(signature[32:])

	c := P256Sm2()
	N := c.Params().N
	one := new(big.Int).SetInt64(1)
	if bigR.Cmp(one) < 0 || bigS.Cmp(one) < 0 {
		return false
	}
	if bigR.Cmp(N) >= 0 || bigS.Cmp(N) >= 0 {
		return false
	}

	t := new(big.Int).Add(bigR, bigS)
	t.Mod(t, N)
	if t.Sign() == 0 {
		return false
	}

	var x *big.Int
	x1, y1 := c.ScalarBaseMult(bigS.Bytes())
	x2, y2 := c.ScalarMult(pub.X, pub.Y, t.Bytes())
	x, _ = c.Add(x1, y1, x2, y2)

	e := new(big.Int).SetBytes(hash)
	x.Add(x, e)
	x.Mod(x, N)
	return x.Cmp(bigR) == 0
}

// VerifyCompressedPubkey verify sigature is valid.
// The parameters is bytes.
// The publickey is compressed, the length is 33 bytes.
func VerifyCompressedPubkey(compressedPublicKey, hash, signature []byte) bool {
	var aa, xx, xx3 sm2P256FieldElement

	P256Sm2()
	x := new(big.Int).SetBytes(compressedPublicKey[1:])
	curve := sm2P256
	sm2P256FromBig(&xx, x)
	sm2P256Square(&xx3, &xx)       // x3 = x ^ 2
	sm2P256Mul(&xx3, &xx3, &xx)    // x3 = x ^ 2 * x
	sm2P256Mul(&aa, &curve.a, &xx) // a = a * x
	sm2P256Add(&xx3, &xx3, &aa)
	sm2P256Add(&xx3, &xx3, &curve.b)

	y2 := sm2P256ToBig(&xx3)
	y := new(big.Int).ModSqrt(y2, sm2P256.P)
	if getLastBit(y) != uint(compressedPublicKey[0]) {
		y.Sub(sm2P256.P, y)
	}

	pub := &PublicKey{
		Curve: P256Sm2(),
		X:     x,
		Y:     y,
	}

	bigR := new(big.Int).SetBytes(signature[:32])
	bigS := new(big.Int).SetBytes(signature[32:])

	c := P256Sm2()
	N := c.Params().N
	one := new(big.Int).SetInt64(1)
	if bigR.Cmp(one) < 0 || bigS.Cmp(one) < 0 {
		return false
	}
	if bigR.Cmp(N) >= 0 || bigS.Cmp(N) >= 0 {
		return false
	}

	t := new(big.Int).Add(bigR, bigS)
	t.Mod(t, N)
	if t.Sign() == 0 {
		return false
	}

	var bigx *big.Int
	x1, y1 := c.ScalarBaseMult(bigS.Bytes())
	x2, y2 := c.ScalarMult(pub.X, pub.Y, t.Bytes())
	bigx, _ = c.Add(x1, y1, x2, y2)

	e := new(big.Int).SetBytes(hash)
	bigx.Add(bigx, e)
	bigx.Mod(bigx, N)
	return bigx.Cmp(bigR) == 0
}
