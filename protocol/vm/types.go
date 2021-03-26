package vm

import (
	"encoding/binary"

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

// Int64Bytes convert int64 to bytes
func Int64Bytes(n int64) []byte {
	if n == 0 {
		return []byte{}
	}
	res := make([]byte, 8)
	// converting int64 to uint64 is a safe operation that
	// preserves all data
	binary.LittleEndian.PutUint64(res, uint64(n))
	for len(res) > 0 && res[len(res)-1] == 0 {
		res = res[:len(res)-1]
	}
	return res
}

// AsInt64 convert bytes to int64
func AsInt64(b []byte) (int64, error) {
	if len(b) == 0 {
		return 0, nil
	}
	if len(b) > 8 {
		return 0, ErrBadValue
	}

	var padded [8]byte
	copy(padded[:], b)

	res := binary.LittleEndian.Uint64(padded[:])
	// converting uint64 to int64 is a safe operation that
	// preserves all data
	return int64(res), nil
}

// BigIntBytes conv big int to bytes, uint256 is version 1.1.1
func BigIntBytes(n *uint256.Int) []byte {
	b := n.Bytes32()
	return b[:]
}

// AsBigInt conv bytes to big int
func AsBigInt(b []byte) (*uint256.Int, error) {
	if len(b) > 32 {
		return nil, ErrBadValue
	}

	return uint256.NewInt().SetBytes(b), nil
}
