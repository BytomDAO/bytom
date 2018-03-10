package difficulty

// HashToBig converts a *bc.Hash into a big.Int that can be used to
import (
	"math/big"

	"github.com/bytom/consensus"
	"github.com/bytom/mining/tensority"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

// HashToBig convert bc.Hash to a difficult int
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

// CheckProofOfWork the hash is vaild for given difficult
func CheckProofOfWork(hash, seed *bc.Hash, bits uint64) bool {
	compareHash := tensority.Hash(hash, seed)
	return HashToBig(compareHash).Cmp(CompactToBig(bits)) <= 0
}

// CalcNextRequiredDifficulty return the difficult for next block
func CalcNextRequiredDifficulty(lastBH, compareBH *legacy.BlockHeader) uint64 {
	if lastBH == nil {
		return consensus.PowMinBits
	} else if (lastBH.Height)%consensus.BlocksPerRetarget != 0 || lastBH.Height == 0 {
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
