package difficulty

import (
	"math/big"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/mining/tensority"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

var (
	// bigOne is 1 represented as a big.Int.  It is defined here to avoid
	// the overhead of creating it multiple times.
	bigOne = big.NewInt(1)

	// oneLsh256 is 1 shifted left 256 bits.  It is defined here to avoid
	// the overhead of creating it multiple times.
	oneLsh256 = new(big.Int).Lsh(bigOne, 256)
)

// HashToBig convert bc.Hash to a difficulty int
func HashToBig(hash *bc.Hash) *big.Int {
	// reverse the bytes of the hash (little-endian) to use it in the big
	// package (big-endian)
	buf := hash.Byte32()
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

// CalcWork calculates a work value from difficulty bits.
func CalcWork(bits uint64) *big.Int {
	difficultyNum := CompactToBig(bits)
	if difficultyNum.Sign() <= 0 {
		return big.NewInt(0)
	}

	// (1 << 256) / (difficultyNum + 1)
	denominator := new(big.Int).Add(difficultyNum, bigOne)
	return new(big.Int).Div(oneLsh256, denominator)
}

// CompactToBig converts a compact representation of a whole unsigned integer
// N to an big.Int. The representation is similar to IEEE754 floating point
// numbers. Sign is not really being used.
//
//	-------------------------------------------------
//	|   Exponent     |    Sign    |    Mantissa     |
//	-------------------------------------------------
//	| 8 bits [63-56] | 1 bit [55] | 55 bits [54-00] |
//	-------------------------------------------------
//
// 	N = (-1^sign) * mantissa * 256^(exponent-3)
//  Actually it will be nicer to use 7 instead of 3 for robustness reason.
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
// an unsigned 64-bit number. Sign is not really being used, but it's kept
// here.
func BigToCompact(n *big.Int) uint64 {
	if n.Sign() == 0 {
		return 0
	}

	var mantissa uint64
	// Bytes() returns the absolute value of n as a big-endian byte slice
	exponent := uint(len(n.Bytes()))

	// Bits() returns the absolute value of n as a little-endian uint64 slice
	if exponent <= 3 {
		mantissa = uint64(n.Bits()[0])
		mantissa <<= 8 * (3 - exponent)
	} else {
		tn := new(big.Int).Set(n)
		// Since the base for the exponent is 256, the exponent can be treated
		// as the number of bytes to represent the full 256-bit number. And as
		// the exponent is treated as the number of bytes, Rsh 8*(exponent-3)
		// makes sure that the shifted tn won't occupy more than 8*3=24 bits,
		// and can be read from Bits()[0], which is 64-bit
		mantissa = uint64(tn.Rsh(tn, 8*(exponent-3)).Bits()[0])
	}

	if mantissa&0x0080000000000000 != 0 {
		mantissa >>= 8
		exponent++
	}

	compact := uint64(exponent)<<56 | mantissa
	if n.Sign() < 0 {
		compact |= 0x0080000000000000
	}
	return compact
}

// CheckProofOfWork checks whether the hash is valid for a given difficulty.
func CheckProofOfWork(hash, seed *bc.Hash, bits uint64) bool {
	compareHash := tensority.AIHash.Hash(hash, seed)
	return HashToBig(compareHash).Cmp(CompactToBig(bits)) <= 0
}

// CalcNextRequiredDifficulty return the difficulty using compact representation
// for next block, when a lower difficulty Int actually reflects a more difficult
// mining progress.
func CalcNextRequiredDifficulty(lastBH, compareBH *types.BlockHeader) uint64 {
	if (lastBH.Height)%consensus.BlocksPerRetarget != 0 || lastBH.Height == 0 {
		return lastBH.Bits
	}

	targetTimeSpan := int64(consensus.BlocksPerRetarget * consensus.TargetSecondsPerBlock)
	actualTimeSpan := int64(lastBH.Timestamp - compareBH.Timestamp)

	oldTarget := CompactToBig(lastBH.Bits)
	newTarget := new(big.Int).Mul(oldTarget, big.NewInt(actualTimeSpan))
	newTarget.Div(newTarget, big.NewInt(targetTimeSpan))
	newTargetBits := BigToCompact(newTarget)

	return newTargetBits
}
