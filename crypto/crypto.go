package crypto

import (
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"

	"github.com/bytom/bytom/common"
	"github.com/bytom/bytom/crypto/sm3"
)

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

func Ripemd160(data []byte) []byte {
	ripemd := ripemd160.New()
	ripemd.Write(data)

	return ripemd.Sum(nil)
}

func Sm3(data ...[]byte) []byte {
	d := sm3.New()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}
