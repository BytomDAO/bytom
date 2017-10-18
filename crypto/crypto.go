// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package crypto

import (
	"github.com/bytom/common"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
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

func Sha3(data ...[]byte) []byte          { return Sha256(data...) }
func Sha3Hash(data ...[]byte) common.Hash { return Sha256Hash(data...) }

func Ripemd160(data []byte) []byte {
	ripemd := ripemd160.New()
	ripemd.Write(data)

	return ripemd.Sum(nil)
}

func PubkeyToAddress(pubBytes []byte) common.Address {
	address, _ := common.AddressEncode("bm", 1, toInt(Ripemd160(Sha3(pubBytes))))
	return common.StringToAddress(address)
}

func AddressToPubkey(addr common.Address) (int, []byte, error) {
	ver, data, err := common.AddressDecode("bm", addr.Str())
	return ver, toBytes(data), err
}

func zeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}

func toInt(bytes []byte) []int {
	ints := make([]int, len(bytes))
	for i := range bytes {
		ints[i] = int(bytes[i])
	}
	return ints
}

func toBytes(ints []int) []byte {
	bytes := make([]byte, len(ints))
	for i := range ints {
		bytes[i] = byte(ints[i])
	}
	return bytes
}
