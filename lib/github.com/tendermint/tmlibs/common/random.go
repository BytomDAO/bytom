package common

import (
	crand "crypto/rand"
	"math/rand"
)

const (
	strChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" // 62 characters
)

func init() {
	b := cRandBytes(8)
	var seed uint64
	for i := 0; i < 8; i++ {
		seed |= uint64(b[i])
		seed <<= 8
	}
	rand.Seed(int64(seed))
}

// Constructs an alphanumeric string of given length.
func RandStr(length int) string {
	chars := []byte{}
MAIN_LOOP:
	for {
		val := rand.Int63()
		for i := 0; i < 10; i++ {
			v := int(val & 0x3f) // rightmost 6 bits
			if v >= 62 {         // only 62 characters in strChars
				val >>= 6
				continue
			} else {
				chars = append(chars, strChars[v])
				if len(chars) == length {
					break MAIN_LOOP
				}
				val >>= 6
			}
		}
	}

	return string(chars)
}

func RandInt() int {
	return rand.Int()
}

func RandBytes(n int) []byte {
	bs := make([]byte, n)
	for i := 0; i < n; i++ {
		bs[i] = byte(rand.Intn(256))
	}
	return bs
}

// NOTE: This relies on the os's random number generator.
// For real security, we should salt that with some seed.
// See github.com/tendermint/go-crypto for a more secure reader.
func cRandBytes(numBytes int) []byte {
	b := make([]byte, numBytes)
	_, err := crand.Read(b)
	if err != nil {
		PanicCrisis(err)
	}
	return b
}
