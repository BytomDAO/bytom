package consensus

import (
	"math/big"

	"github.com/bytom/protocol/bc"
)

const (
	subsidyReductionInterval = uint64(560640)
	baseSubsidy              = uint64(624000000000)
)

// HashToBig converts a *bc.Hash into a big.Int that can be used to
// perform math comparisons.
func HashToBig(hash *bc.Hash) *big.Int {
	buf := hash.Byte32()
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

// CompactToBig converts a compact representation of a whole number N to an
// unsigned 64-bit number.  The representation is similar to IEEE754 floating
// point numbers.
//
//	-------------------------------------------------
//	|   Exponent     |    Sign    |    Mantissa     |
//	-------------------------------------------------
//	| 8 bits [63-56] | 1 bit [55] | 55 bits [54-00] |
//	-------------------------------------------------
//
// 	N = (-1^sign) * mantissa * 256^(exponent-3)
func CompactToBig(compact uint64) *big.Int {
	// Extract the mantissa, sign bit, and exponent.
	mantissa := compact & 0x007fffffffffffff
	isNegative := compact&0x0080000000000000 != 0
	exponent := uint(compact >> 56)

	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}

// BigToCompact converts a whole number N to a compact representation using
// an unsigned 64-bit number
func BigToCompact(n *big.Int) uint64 {
	if n.Sign() == 0 {
		return 0
	}

	var mantissa uint64
	exponent := uint(len(n.Bytes()))
	if exponent <= 3 {
		mantissa = uint64(n.Bits()[0])
		mantissa <<= 8 * (3 - exponent)
	} else {
		tn := new(big.Int).Set(n)
		mantissa = uint64(tn.Rsh(tn, 8*(exponent-3)).Bits()[0])
	}

	if mantissa&0x0080000000000000 != 0 {
		mantissa >>= 8
		exponent++
	}

	compact := uint64(exponent<<56) | mantissa
	if n.Sign() < 0 {
		compact |= 0x0080000000000000
	}
	return compact
}

func CheckProofOfWork(hash *bc.Hash, bits uint64) bool {
	if HashToBig(hash).Cmp(CompactToBig(bits)) <= 0 {
		return true
	}
	return false
}

func CalcNextRequiredDifficulty() uint64 {
	return uint64(2161727821138738707)
}

func BlockSubsidy(height uint64) uint64 {
	return baseSubsidy >> uint(height/subsidyReductionInterval)
}
