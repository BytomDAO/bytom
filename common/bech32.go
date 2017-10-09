// Package bech32 reference implementation for Bech32 and segwit addresses.
// Copyright (c) 2017 Takatoshi Nakagawa
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
package common

import (
	"bytes"
	"fmt"
	"strings"
)

var charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

var generator = []int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}

func polymod(values []int) int {
	chk := 1
	for _, v := range values {
		top := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ v
		for i := 0; i < 5; i++ {
			if (top>>uint(i))&1 == 1 {
				chk ^= generator[i]
			}
		}
	}
	return chk
}

func hrpExpand(hrp string) []int {
	ret := []int{}
	for _, c := range hrp {
		ret = append(ret, int(c>>5))
	}
	ret = append(ret, 0)
	for _, c := range hrp {
		ret = append(ret, int(c&31))
	}
	return ret
}

func verifyChecksum(hrp string, data []int) bool {
	return polymod(append(hrpExpand(hrp), data...)) == 1
}

func createChecksum(hrp string, data []int) []int {
	values := append(append(hrpExpand(hrp), data...), []int{0, 0, 0, 0, 0, 0}...)
	mod := polymod(values) ^ 1
	ret := make([]int, 6)
	for p := 0; p < len(ret); p++ {
		ret[p] = (mod >> uint(5*(5-p))) & 31
	}
	return ret
}

// Encode encodes hrp(human-readable part) and data(32bit data array), returns Bech32 / or error
// if hrp is uppercase, return uppercase Bech32
func Encode(hrp string, data []int) (string, error) {
	if (len(hrp) + len(data) + 7) > 90 {
		return "", fmt.Errorf("too long : hrp length=%d, data length=%d", len(hrp), len(data))
	}
	if len(hrp) < 1 {
		return "", fmt.Errorf("invalid hrp : hrp=%v", hrp)
	}
	for p, c := range hrp {
		if c < 33 || c > 126 {
			return "", fmt.Errorf("invalid character human-readable part : hrp[%d]=%d", p, c)
		}
	}
	if strings.ToUpper(hrp) != hrp && strings.ToLower(hrp) != hrp {
		return "", fmt.Errorf("mix case : hrp=%v", hrp)
	}
	lower := strings.ToLower(hrp) == hrp
	hrp = strings.ToLower(hrp)
	combined := append(data, createChecksum(hrp, data)...)
	var ret bytes.Buffer
	ret.WriteString(hrp)
	ret.WriteString("1")
	for idx, p := range combined {
		if p < 0 || p >= len(charset) {
			return "", fmt.Errorf("invalid data : data[%d]=%d", idx, p)
		}
		ret.WriteByte(charset[p])
	}
	if lower {
		return ret.String(), nil
	}
	return strings.ToUpper(ret.String()), nil
}

// Decode decodes bechString(Bech32) returns hrp(human-readable part) and data(32bit data array) / or error
func Decode(bechString string) (string, []int, error) {
	if len(bechString) > 90 {
		return "", nil, fmt.Errorf("too long : len=%d", len(bechString))
	}
	if strings.ToLower(bechString) != bechString && strings.ToUpper(bechString) != bechString {
		return "", nil, fmt.Errorf("mixed case")
	}
	bechString = strings.ToLower(bechString)
	pos := strings.LastIndex(bechString, "1")
	if pos < 1 || pos+7 > len(bechString) {
		return "", nil, fmt.Errorf("separator '1' at invalid position : pos=%d , len=%d", pos, len(bechString))
	}
	hrp := bechString[0:pos]
	for p, c := range hrp {
		if c < 33 || c > 126 {
			return "", nil, fmt.Errorf("invalid character human-readable part : bechString[%d]=%d", p, c)
		}
	}
	data := []int{}
	for p := pos + 1; p < len(bechString); p++ {
		d := strings.Index(charset, fmt.Sprintf("%c", bechString[p]))
		if d == -1 {
			return "", nil, fmt.Errorf("invalid character data part : bechString[%d]=%d", p, bechString[p])
		}
		data = append(data, d)
	}
	if !verifyChecksum(hrp, data) {
		return "", nil, fmt.Errorf("invalid checksum")
	}
	return hrp, data[:len(data)-6], nil
}

func convertbits(data []int, frombits, tobits uint, pad bool) ([]int, error) {
	acc := 0
	bits := uint(0)
	ret := []int{}
	maxv := (1 << tobits) - 1
	for idx, value := range data {
		if value < 0 || (value>>frombits) != 0 {
			return nil, fmt.Errorf("invalid data range : data[%d]=%d (frombits=%d)", idx, value, frombits)
		}
		acc = (acc << frombits) | value
		bits += frombits
		for bits >= tobits {
			bits -= tobits
			ret = append(ret, (acc>>bits)&maxv)
		}
	}
	if pad {
		if bits > 0 {
			ret = append(ret, (acc<<(tobits-bits))&maxv)
		}
	} else if bits >= frombits {
		return nil, fmt.Errorf("illegal zero padding")
	} else if ((acc << (tobits - bits)) & maxv) != 0 {
		return nil, fmt.Errorf("non-zero padding")
	}
	return ret, nil
}

// AddressDecode decodes hrp(human-readable part) Address(string), returns version(int) and data(bytes array) / or error
func AddressDecode(hrp, addr string) (int, []int, error) {
	dechrp, data, err := Decode(addr)
	if err != nil {
		return -1, nil, err
	}
	if dechrp != hrp {
		return -1, nil, fmt.Errorf("invalid human-readable part : %s != %s", hrp, dechrp)
	}
	if len(data) < 1 {
		return -1, nil, fmt.Errorf("invalid decode data length : %d", len(data))
	}
	if data[0] > 16 {
		return -1, nil, fmt.Errorf("invalid address version : %d", data[0])
	}
	res, err := convertbits(data[1:], 5, 8, false)
	if err != nil {
		return -1, nil, err
	}
	if len(res) < 2 || len(res) > 40 {
		return -1, nil, fmt.Errorf("invalid convertbits length : %d", len(res))
	}
	if data[0] == 0 && len(res) != 20 && len(res) != 32 {
		return -1, nil, fmt.Errorf("invalid program length for witness version 0 (per BIP141) : %d", len(res))
	}
	return data[0], res, nil
}

// AddressEncode encodes hrp(human-readable part) , version(int) and data(bytes array), returns  Address / or error
func AddressEncode(hrp string, version int, pubkey []int) (string, error) {
	if version < 0 || version > 16 {
		return "", fmt.Errorf("invalid version : %d", version)
	}
	if len(pubkey) < 2 || len(pubkey) > 40 {
		return "", fmt.Errorf("invalid pubkey hash length : %d", len(pubkey))
	}
	if version == 0 && len(pubkey) != 20 && len(pubkey) != 32 {
		return "", fmt.Errorf("invalid program length for witness version 0 (per BIP141) : %d", len(pubkey))
	}
	data, err := convertbits(pubkey, 8, 5, true)
	if err != nil {
		return "", err
	}
	ret, err := Encode(hrp, append([]int{version}, data...))
	if err != nil {
		return "", err
	}
	return ret, nil
}
