package common

import (
	_ "encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"math/rand"
	"reflect"
	"strings"
)

const (
	HashLength       = 32
	AddressLength    = 42
	PubkeyHashLength = 20
)

var hashJsonLengthErr = errors.New("common: unmarshalJSON failed: hash must be exactly 32 bytes")

type (
	Hash [HashLength]byte
)

func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

func StringToHash(s string) Hash { return BytesToHash([]byte(s)) }
func BigToHash(b *big.Int) Hash  { return BytesToHash(b.Bytes()) }

// Don't use the default 'String' method in case we want to overwrite

// Get the string representation of the underlying hash
func (h Hash) Str() string   { return string(h[:]) }
func (h Hash) Bytes() []byte { return h[:] }
func (h Hash) Hex() string   { return "0x" + Bytes2Hex(h[:]) }

// UnmarshalJSON parses a hash in its hex from to a hash.
func (h *Hash) UnmarshalJSON(input []byte) error {
	length := len(input)
	if length >= 2 && input[0] == '"' && input[length-1] == '"' {
		input = input[1 : length-1]
	}
	// strip "0x" for length check
	if len(input) > 1 && strings.ToLower(string(input[:2])) == "0x" {
		input = input[2:]
	}

	// validate the length of the input hash
	if len(input) != HashLength*2 {
		return hashJsonLengthErr
	}
	h.SetBytes(FromHex(string(input)))
	return nil
}

// Serialize given hash to JSON
func (h Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Hex())
}

// Sets the hash to the value of b. If b is larger than len(h) it will panic
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// Sets h to other
func (h *Hash) Set(other Hash) {
	for i, v := range other {
		h[i] = v
	}
}

// Generate implements testing/quick.Generator.
func (h Hash) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(len(h))
	for i := len(h) - 1; i > m; i-- {
		h[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(h)
}

func EmptyHash(h Hash) bool {
	return h == Hash{}
}
