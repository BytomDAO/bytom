// Package ed25519 implements the Ed25519 signature algorithm. See
// http://ed25519.cr.yp.to/.
//
// These functions are also compatible with the “Ed25519” function defined in
// https://tools.ietf.org/html/draft-irtf-cfrg-eddsa-05.
package ed25519

// This code is a port of the public domain, “ref10” implementation of ed25519
// from SUPERCOP.

import (
	cryptorand "crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"strconv"

	"github.com/bytom/bytom/crypto/ed25519/internal/edwards25519"
)

const (
	// PublicKeySize is the size, in bytes, of public keys as used in this package.
	PublicKeySize = 32
	// PrivateKeySize is the size, in bytes, of private keys as used in this package.
	PrivateKeySize = 64
	// SignatureSize is the size, in bytes, of signatures generated and verified by this package.
	SignatureSize = 64
	// SeedSize is the size, in bytes, of private key seeds. These are the private key representations used by RFC 8032.
	SeedSize = 32
)

// PublicKey is the type of Ed25519 public keys.
type PublicKey []byte

// PrivateKey is the type of Ed25519 private keys. It implements crypto.Signer.
type PrivateKey []byte

// Public returns the PublicKey corresponding to priv.
func (priv PrivateKey) Public() PublicKey {
	publicKey := make([]byte, PublicKeySize)
	copy(publicKey, priv[32:])
	return PublicKey(publicKey)
}

// Seed returns the private key seed corresponding to priv. It is provided for
// interoperability with RFC 8032. RFC 8032's private keys correspond to seeds
// in this package.
func (priv PrivateKey) Seed() []byte {
	seed := make([]byte, SeedSize)
	copy(seed, priv[:32])
	return seed
}

func (priv PrivateKey) String() string {
	return hex.EncodeToString(priv)
}

// GenerateKey generates a public/private key pair using entropy from rand.
// If rand is nil, crypto/rand.Reader will be used.
func GenerateKey(rand io.Reader) (publicKey PublicKey, privateKey PrivateKey, err error) {
	if rand == nil {
		rand = cryptorand.Reader
	}

	privateKey = make([]byte, PrivateKeySize)
	publicKey = make([]byte, PublicKeySize)
	_, err = io.ReadFull(rand, privateKey[:32])
	if err != nil {
		return nil, nil, err
	}

	digest := sha512.Sum512(privateKey[:32])
	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	var A edwards25519.ExtendedGroupElement
	var hBytes [32]byte
	copy(hBytes[:], digest[:])
	edwards25519.GeScalarMultBase(&A, &hBytes)
	var publicKeyBytes [32]byte
	A.ToBytes(&publicKeyBytes)

	copy(privateKey[32:], publicKeyBytes[:])
	copy(publicKey, publicKeyBytes[:])

	return publicKey, privateKey, nil
}

// NewKeyFromSeed calculates a private key from a seed. It will panic if
// len(seed) is not SeedSize. This function is provided for interoperability
// with RFC 8032. RFC 8032's private keys correspond to seeds in this
// package.
func NewKeyFromSeed(seed []byte) PrivateKey {
	if l := len(seed); l != SeedSize {
		panic("ed25519: bad seed length: " + strconv.Itoa(l))
	}

	digest := sha512.Sum512(seed)
	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	var A edwards25519.ExtendedGroupElement
	var hBytes [32]byte
	copy(hBytes[:], digest[:])
	edwards25519.GeScalarMultBase(&A, &hBytes)
	var publicKeyBytes [32]byte
	A.ToBytes(&publicKeyBytes)

	privateKey := make([]byte, PrivateKeySize)
	copy(privateKey, seed)
	copy(privateKey[32:], publicKeyBytes[:])

	return privateKey
}

// Sign signs the message with privateKey and returns a signature. It will
// panic if len(privateKey) is not PrivateKeySize.
func Sign(privateKey PrivateKey, message []byte) []byte {
	if l := len(privateKey); l != PrivateKeySize {
		panic("ed25519: bad private key length: " + strconv.Itoa(l))
	}

	h := sha512.New()
	h.Write(privateKey[:32])

	var digest1, messageDigest, hramDigest [64]byte
	var expandedSecretKey [32]byte
	h.Sum(digest1[:0])
	copy(expandedSecretKey[:], digest1[:])
	expandedSecretKey[0] &= 248
	expandedSecretKey[31] &= 63
	expandedSecretKey[31] |= 64

	h.Reset()
	h.Write(digest1[32:])
	h.Write(message)
	h.Sum(messageDigest[:0])

	var messageDigestReduced [32]byte
	edwards25519.ScReduce(&messageDigestReduced, &messageDigest)
	var R edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&R, &messageDigestReduced)

	var encodedR [32]byte
	R.ToBytes(&encodedR)

	h.Reset()
	h.Write(encodedR[:])
	h.Write(privateKey[32:])
	h.Write(message)
	h.Sum(hramDigest[:0])
	var hramDigestReduced [32]byte
	edwards25519.ScReduce(&hramDigestReduced, &hramDigest)

	var s [32]byte
	edwards25519.ScMulAdd(&s, &hramDigestReduced, &expandedSecretKey, &messageDigestReduced)

	signature := make([]byte, SignatureSize)
	copy(signature[:], encodedR[:])
	copy(signature[32:], s[:])

	return signature
}

// Verify reports whether sig is a valid signature of message by publicKey. It
// will panic if len(publicKey) is not PublicKeySize.
func Verify(publicKey PublicKey, message, sig []byte) bool {
	if l := len(publicKey); l != PublicKeySize {
		panic("ed25519: bad public key length: " + strconv.Itoa(l))
	}

	if len(sig) != SignatureSize || sig[63]&224 != 0 {
		return false
	}

	var A edwards25519.ExtendedGroupElement
	var publicKeyBytes [32]byte
	copy(publicKeyBytes[:], publicKey)
	if !A.FromBytes(&publicKeyBytes) {
		return false
	}
	edwards25519.FeNeg(&A.X, &A.X)
	edwards25519.FeNeg(&A.T, &A.T)

	h := sha512.New()
	h.Write(sig[:32])
	h.Write(publicKey[:])
	h.Write(message)
	var digest [64]byte
	h.Sum(digest[:0])

	var hReduced [32]byte
	edwards25519.ScReduce(&hReduced, &digest)

	var R edwards25519.ProjectiveGroupElement
	var b [32]byte
	copy(b[:], sig[32:])
	edwards25519.GeDoubleScalarMultVartime(&R, &hReduced, &A, &b)

	var checkR [32]byte
	R.ToBytes(&checkR)
	return subtle.ConstantTimeCompare(sig[:32], checkR[:]) == 1
}
