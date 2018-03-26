package difficulty

import (
	"math/big"
	"testing"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc/types"
)

func TestCalcNextRequiredDifficulty(t *testing.T) {
	targetTimeSpan := uint64(consensus.BlocksPerRetarget * consensus.TargetSecondsPerBlock)
	cases := []struct {
		lastBH    *types.BlockHeader
		compareBH *types.BlockHeader
		want      uint64
	}{
		//{nil, nil, powMinBits},
		//{&types.BlockHeader{Height: BlocksPerRetarget, Bits: 87654321}, nil, 87654321},
		{
			&types.BlockHeader{Height: consensus.BlocksPerRetarget, Timestamp: targetTimeSpan, Bits: BigToCompact(big.NewInt(1000))},
			&types.BlockHeader{Height: 0, Timestamp: 0},
			BigToCompact(big.NewInt(1000)),
		},
		{
			&types.BlockHeader{Height: consensus.BlocksPerRetarget, Timestamp: targetTimeSpan * 2, Bits: BigToCompact(big.NewInt(1000))},
			&types.BlockHeader{Height: 0, Timestamp: 0},
			BigToCompact(big.NewInt(2000)),
		},
		{
			&types.BlockHeader{Height: consensus.
				BlocksPerRetarget, Timestamp: targetTimeSpan / 2, Bits: BigToCompact(big.NewInt(1000))},
			&types.BlockHeader{Height: 0, Timestamp: 0},
			BigToCompact(big.NewInt(500)),
		},
	}

	for i, c := range cases {
		if got := CalcNextRequiredDifficulty(c.lastBH, c.compareBH); got != c.want {
			t.Errorf("Compile(%d) = %d want %d", i, got, c.want)
		}
	}
}
