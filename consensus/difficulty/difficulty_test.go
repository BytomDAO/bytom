package difficulty

import (
	"math/big"
	"testing"
	"reflect"
	"strconv"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/bc"
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

func TestHashToBig(t *testing.T) {
	cases := []struct {
		hashBytes	[32]byte
		expect		[32]byte
	}{
		{
			hashBytes: 	[32]byte{
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
			},
			expect: 	[32]byte{
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
			},
		},
		{
			hashBytes: 	[32]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
				0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
			},
			expect: 	[32]byte{
				0x0f, 0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09, 0x08,
				0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			hashBytes: 	[32]byte{
				0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7,
				0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xef,
				0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7,
				0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff,
			},
			expect: 	[32]byte{
				0xff, 0xfe, 0xfd, 0xfc, 0xfb, 0xfa, 0xf9, 0xf8,
				0xf7, 0xf6, 0xf5, 0xf4, 0xf3, 0xf2, 0xf1, 0xf0,
				0xef, 0xee, 0xed, 0xec, 0xeb, 0xea, 0xe9, 0xe8,
				0xe7, 0xe6, 0xe5, 0xe4, 0xe3, 0xe2, 0xe1, 0xe0,
			},
		},
	}

	for i, c := range cases {
		bhash := bc.NewHash(c.hashBytes)

		result := HashToBig(&bhash).Bytes()

		var resArr [32]byte
		copy(resArr[:], result)

		if !reflect.DeepEqual(resArr, c.expect) {
			t.Errorf("case %d: content mismatch:\n\tgeting\t\t%x\n\texpecting\t%x", i, resArr, c.expect)
		}
	}
}

func TestCompactToBig(t *testing.T) {
	cases := []struct {
		BStrCompact 	string
		expect			int64
	}{
		{
			BStrCompact:	`00000011` + //Exponent
							`0` + //Sign
							`0000000000000000000000000000000000000000000000000000000`, //
			expect:	 		0,
		},
		{
			BStrCompact:	`00000011` + //Exponent
							`1` + //Sign
							`0000000000000000000000000000000000000000000000000000000`, //
			expect:	 		0,
		},
		{
			BStrCompact:	`00000011` + //Exponent
							`0` + //Sign
							`0000000000000000000000000000000000000000000000000000001`, //
			expect:	 		1,
		},
		{
			BStrCompact:	`00000011` + //Exponent
							`1` + //Sign
							`0000000000000000000000000000000000000000000000000000001`, //
			expect:	 		-1,
		},
	}

	for i, c := range cases {
		compact, _ := strconv.ParseUint(c.BStrCompact, 2, 64)

		result := CompactToBig(compact).Int64()

		if result != c.expect {
			t.Errorf("case %d: content mismatch:\n\tgeting\t\t%x\n\texpecting\t%x", i, result, c.expect)
		}
	}
}
