package crypto

import (
	"github.com/bytom/common"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
)

func DoubleSha256(b []byte) []byte {
	hasher := sha3.New256()
	hasher.Write(b)
	sum := hasher.Sum(nil)
	hasher.Reset()
	hasher.Write(sum)
	return hasher.Sum(nil)
}

func Sha256(data ...[]byte) []byte {
	d := sha3.New256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

func Sha256Hash(data ...[]byte) (h common.Hash) {
	d := sha3.New256()
	for _, b := range data {
		d.Write(b)
	}
	d.Sum(h[:0])
	return h
}

func Sha3(data ...[]byte) []byte          { return Sha256(data...) }
func Sha3Hash(data ...[]byte) common.Hash { return Sha256Hash(data...) }

func Ripemd160(data []byte) []byte {
	ripemd := ripemd160.New()
	ripemd.Write(data)

	return ripemd.Sum(nil)
}
