package consensus

// HashToBig converts a *bc.Hash into a big.Int that can be used to
import (
	"math/big"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

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
	return HashToBig(hash).Cmp(CompactToBig(bits)) <= 0
}

func CalcNextRequiredDifficulty(lastBH, compareBH *legacy.BlockHeader) uint64 {
	if lastBH == nil {
		return powMinBits
	} else if (lastBH.Height+1)%BlocksPerRetarget != 0 {
		return lastBH.Bits
	}

	targetTimeSpan := int64(BlocksPerRetarget * targetSecondsPerBlock)
	actualTimespan := int64(lastBH.Time().Sub(compareBH.Time()).Seconds())

	oldTarget := CompactToBig(lastBH.Bits)
	newTarget := new(big.Int).Mul(oldTarget, big.NewInt(actualTimespan))
	newTarget.Div(newTarget, big.NewInt(targetTimeSpan))
	newTargetBits := BigToCompact(newTarget)

	return newTargetBits
}
