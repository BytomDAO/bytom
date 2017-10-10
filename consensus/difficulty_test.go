package consensus

import (
	"math/big"
	"testing"

	"github.com/bytom/protocol/bc/legacy"
)

func TestCalcNextRequiredDifficulty(t *testing.T) {
	targetTimeSpan := uint64(BlocksPerRetarget * targetSecondsPerBlock * 1000)
	cases := []struct {
		lastBH    *legacy.BlockHeader
		compareBH *legacy.BlockHeader
		want      uint64
	}{
		{nil, nil, powMinBits},
		{&legacy.BlockHeader{Height: BlocksPerRetarget, Bits: 87654321}, nil, 87654321},
		{
			&legacy.BlockHeader{Height: BlocksPerRetarget - 1, TimestampMS: targetTimeSpan, Bits: BigToCompact(big.NewInt(1000))},
			&legacy.BlockHeader{Height: 0, TimestampMS: 0},
			BigToCompact(big.NewInt(1000)),
		},
		{
			&legacy.BlockHeader{Height: BlocksPerRetarget - 1, TimestampMS: targetTimeSpan * 2, Bits: BigToCompact(big.NewInt(1000))},
			&legacy.BlockHeader{Height: 0, TimestampMS: 0},
			BigToCompact(big.NewInt(2000)),
		},
		{
			&legacy.BlockHeader{Height: BlocksPerRetarget - 1, TimestampMS: targetTimeSpan / 2, Bits: BigToCompact(big.NewInt(1000))},
			&legacy.BlockHeader{Height: 0, TimestampMS: 0},
			BigToCompact(big.NewInt(500)),
		},
	}

	for i, c := range cases {
		if got := CalcNextRequiredDifficulty(c.lastBH, c.compareBH); got != c.want {
			t.Errorf("Compile(%d) = %d want %d", i, got, c.want)
		}
	}
}
