package vm

import (
	"github.com/holiman/uint256"
)

var trueBytes = []byte{1}

// BoolBytes convert bool to bytes
func BoolBytes(b bool) (result []byte) {
	if b {
		return trueBytes
	}
	return []byte{}
}

// AsBool convert bytes to bool
func AsBool(bytes []byte) bool {
	for _, b := range bytes {
		if b != 0 {
			return true
		}
	}
	return false
}

// Uint64Bytes convert uint64 to bytes in vm
func Uint64Bytes(n uint64) []byte {
	return BigIntBytes(uint256.NewInt(n))
}

// BigIntBytes conv big int to little endian bytes, uint256 is version 1.1.1
func BigIntBytes(n *uint256.Int) []byte {
	return reverse(n.Bytes())
}

// AsBigInt conv little endian bytes to big int
func AsBigInt(b []byte) (*uint256.Int, error) {
	if len(b) > 32 {
		return nil, ErrBadValue
	}

	res := uint256.NewInt(0).SetBytes(reverse(b))
	if res.Sign() < 0 {
		return nil, ErrRange
	}

	return res, nil
}

func bigIntInt64(n *uint256.Int) (int64, error) {
	if !n.IsUint64() {
		return 0, ErrBadValue
	}

	i := int64(n.Uint64())
	if i < 0 {
		return 0, ErrBadValue
	}
	return i, nil
}

// reverse []byte.
func reverse(b []byte) []byte {
	r := make([]byte, len(b))
	copy(r, b)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return r
}
