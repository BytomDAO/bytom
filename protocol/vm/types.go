package vm

import (
	"encoding/binary"
	"math/big"
)

const intGobVersion byte = 1

var trueBytes = []byte{1}

func BoolBytes(b bool) (result []byte) {
	if b {
		return trueBytes
	}
	return []byte{}
}

func AsBool(bytes []byte) bool {
	for _, b := range bytes {
		if b != 0 {
			return true
		}
	}
	return false
}

// todo
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

func BigIntBytes(n *big.Int) []byte {
	res := n.Bytes()
	b := intGobVersion << 1 // make space for sign bit
	if n.Sign() < 0 {
		b |= 1
	}
	res = append(res, b)
	return res
}

func AsBigInt(b []byte) (*big.Int, error) {
	if len(b) == 0 {
		return new(big.Int).SetInt64(0), nil
	}

	buf := b[len(b)-1]
	if buf>>1 != intGobVersion {
		return nil, ErrBadValue
	}
	var res = new(big.Int)
	res.SetBytes(b[0 : len(b)-1])
	if buf&1 != 0 {
		res.Neg(res)
	}

	return res, nil
}

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

func BigIntBytes1(n *big.Int) []byte {
	// MarshalText return ([]byte,error) and error always is nil
	bytes, _ := n.MarshalText()
	return bytes
}

func AsBigInt1(b []byte) (*big.Int, error) {
	res := new(big.Int)
	if err := res.UnmarshalText(b); err != nil {
		return nil, ErrBadValue
	}

	return res, nil
}
