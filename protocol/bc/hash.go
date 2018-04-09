package bc

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/sha3"
)

// EmptyStringHash represents a 256-bit hash.
var EmptyStringHash = NewHash(sha3.Sum256(nil))

// NewHash convert the input byte array to hash
func NewHash(b32 [32]byte) (h Hash) {
	h.V0 = binary.BigEndian.Uint64(b32[0:8])
	h.V1 = binary.BigEndian.Uint64(b32[8:16])
	h.V2 = binary.BigEndian.Uint64(b32[16:24])
	h.V3 = binary.BigEndian.Uint64(b32[24:32])
	return h
}

// Byte32 return the byte array representation
func (h Hash) Byte32() (b32 [32]byte) {
	binary.BigEndian.PutUint64(b32[0:8], h.V0)
	binary.BigEndian.PutUint64(b32[8:16], h.V1)
	binary.BigEndian.PutUint64(b32[16:24], h.V2)
	binary.BigEndian.PutUint64(b32[24:32], h.V3)
	return b32
}

// MarshalText satisfies the TextMarshaler interface.
// It returns the bytes of h encoded in hex,
// for formats that can't hold arbitrary binary data.
// It never returns an error.
func (h Hash) MarshalText() ([]byte, error) {
	b := h.Byte32()
	v := make([]byte, 64)
	hex.Encode(v, b[:])
	return v, nil
}

// UnmarshalText satisfies the TextUnmarshaler interface.
// It decodes hex data from b into h.
func (h *Hash) UnmarshalText(v []byte) error {
	var b [32]byte
	if len(v) != 64 {
		return fmt.Errorf("bad length hash string %d", len(v))
	}
	_, err := hex.Decode(b[:], v)
	*h = NewHash(b)
	return err
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
// If b is a JSON-encoded null, it copies the zero-value into h. Othwerwise, it
// decodes hex data from b into h.
func (h *Hash) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		*h = Hash{}
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return h.UnmarshalText([]byte(s))
}

// Bytes returns the byte representation
func (h Hash) Bytes() []byte {
	b32 := h.Byte32()
	return b32[:]
}

// WriteTo satisfies the io.WriterTo interface.
func (h Hash) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(h.Bytes())
	return int64(n), err
}

// ReadFrom satisfies the io.ReaderFrom interface.
func (h *Hash) ReadFrom(r io.Reader) (int64, error) {
	var b32 [32]byte
	n, err := io.ReadFull(r, b32[:])
	if err != nil {
		return int64(n), err
	}
	*h = NewHash(b32)
	return int64(n), nil
}

// IsZero tells whether a Hash pointer is nil or points to an all-zero
// hash.
func (h *Hash) IsZero() bool {
	if h == nil {
		return true
	}
	return *h == Hash{}
}
