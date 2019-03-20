package difficulty

import (
	"math/big"
	"reflect"
	"strconv"
	"testing"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// A lower difficulty Int actually reflects a more difficult mining progress.
func TestCalcNextRequiredDifficulty(t *testing.T) {
	targetTimeSpan := uint64(consensus.BlocksPerRetarget * consensus.TargetSecondsPerBlock)
	cases := []struct {
		lastBH    *types.BlockHeader
		compareBH *types.BlockHeader
		want      uint64
	}{
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0,
			},
			BigToCompact(big.NewInt(1000)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan * 2,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0,
			},
			BigToCompact(big.NewInt(2000)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: targetTimeSpan*2 - consensus.TargetSecondsPerBlock,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0,
			},
			BigToCompact(big.NewInt(1000)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan / 2,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0,
			},
			BigToCompact(big.NewInt(500)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget * 2,
				Timestamp: targetTimeSpan + targetTimeSpan*2,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan,
			},
			BigToCompact(big.NewInt(2000)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget * 2,
				Timestamp: targetTimeSpan + targetTimeSpan/2,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan,
			},
			BigToCompact(big.NewInt(500)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget*2 - 1,
				Timestamp: targetTimeSpan + targetTimeSpan*2 - consensus.TargetSecondsPerBlock,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan,
			},
			BigToCompact(big.NewInt(1000)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget*2 - 1,
				Timestamp: targetTimeSpan + targetTimeSpan/2 - consensus.TargetSecondsPerBlock,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan,
			},
			BigToCompact(big.NewInt(1000)),
		},
		// lastBH.Height: 0, lastBH.Timestamp - compareBH.Timestamp: 0, lastBH.Bits: 0
		{
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0,
				Bits:      0,
			},
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0,
			},
			0,
		},
		// lastBH.Height: 0, lastBH.Timestamp - compareBH.Timestamp: 0, lastBH.Bits: 18446744073709551615
		{
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0,
				Bits:      18446744073709551615,
			},
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0,
			},
			18446744073709551615,
		},
		// lastBH.Height: 0, lastBH.Timestamp - compareBH.Timestamp: 0, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0,
			},
			BigToCompact(big.NewInt(1000)),
		},
		// lastBH.Height: consensus.BlocksPerRetarget, lastBH.Timestamp - compareBH.Timestamp: 0, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: targetTimeSpan,
			},
			0,
		},
		// lastBH.Height: consensus.BlocksPerRetarget, lastBH.Timestamp - compareBH.Timestamp: -9223372036854775808, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: targetTimeSpan + 9223372036854775808,
			},
			540431955291560988,
		},
		// lastBH.Height: consensus.BlocksPerRetarget, lastBH.Timestamp - compareBH.Timestamp: 9223372036854775807, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan + 9223372036854775807,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: targetTimeSpan,
			},
			504403158272597019,
		},
		// lastBH.Height: consensus.BlocksPerRetarget, lastBH.Timestamp - compareBH.Timestamp: 18446744073709551615, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: 18446744073709551615,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: 0,
			},
			108086391056957440,
		},
		// lastBH.Height: consensus.BlocksPerRetarget, lastBH.Timestamp - compareBH.Timestamp: 302400, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan * 2,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: targetTimeSpan,
			},
			BigToCompact(big.NewInt(1000)),
		},
		// lastBH.Height: consensus.BlocksPerRetarget, lastBH.Timestamp - compareBH.Timestamp: 604800, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan * 3,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: targetTimeSpan,
			},
			144115188076367872,
		},
		// lastBH.Height: consensus.BlocksPerRetarget, lastBH.Timestamp - compareBH.Timestamp: 151200, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan + 9223372036854775807,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: targetTimeSpan,
			},
			504403158272597019,
		},
		// lastBH.Height: consensus.BlocksPerRetarget, lastBH.Timestamp - compareBH.Timestamp: 302400, lastBH.Bits: 0
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan * 2,
				Bits:      0,
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: targetTimeSpan,
			},
			0,
		},
		// lastBH.Height: consensus.BlocksPerRetarget, lastBH.Timestamp - compareBH.Timestamp: 302400, lastBH.Bits: 18446744073709551615
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan * 2,
				Bits:      18446744073709551615,
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: targetTimeSpan,
			},
			252201579141136384,
		},
		// lastBH.Height: consensus.BlocksPerRetarget + 1, lastBH.Timestamp - compareBH.Timestamp: 302400, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget + 1,
				Timestamp: targetTimeSpan * 2,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan,
			},
			BigToCompact(big.NewInt(1000)),
		},
		// lastBH.Height: consensus.BlocksPerRetarget - 1, lastBH.Timestamp - compareBH.Timestamp: 302400, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: targetTimeSpan * 2,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 2,
				Timestamp: targetTimeSpan,
			},
			BigToCompact(big.NewInt(1000)),
		},
		// lastBH.Height: consensus.BlocksPerRetarget * 2, lastBH.Timestamp - compareBH.Timestamp: 302400, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget * 2,
				Timestamp: targetTimeSpan * 2,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget*2 - 1,
				Timestamp: targetTimeSpan,
			},
			BigToCompact(big.NewInt(1000)),
		},
		// lastBH.Height: consensus.BlocksPerRetarget / 2, lastBH.Timestamp - compareBH.Timestamp: 302400, lastBH.Bits: bigInt(1000)
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget / 2,
				Timestamp: targetTimeSpan * 2,
				Bits:      BigToCompact(big.NewInt(1000)),
			},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget/2 - 1,
				Timestamp: targetTimeSpan,
			},
			BigToCompact(big.NewInt(1000)),
		},
	}

	for i, c := range cases {
		if got := CalcNextRequiredDifficulty(c.lastBH, c.compareBH); got != c.want {
			t.Errorf("Compile(%d) = %d want %d\n", i, got, c.want)
			return
		}
	}
}

func TestHashToBig(t *testing.T) {
	cases := []struct {
		in  [32]byte
		out [32]byte
	}{
		{
			in: [32]byte{
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
			},
			out: [32]byte{
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
			},
		},
		{
			in: [32]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
				0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
			},
			out: [32]byte{
				0x0f, 0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09, 0x08,
				0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			in: [32]byte{
				0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7,
				0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xef,
				0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7,
				0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff,
			},
			out: [32]byte{
				0xff, 0xfe, 0xfd, 0xfc, 0xfb, 0xfa, 0xf9, 0xf8,
				0xf7, 0xf6, 0xf5, 0xf4, 0xf3, 0xf2, 0xf1, 0xf0,
				0xef, 0xee, 0xed, 0xec, 0xeb, 0xea, 0xe9, 0xe8,
				0xe7, 0xe6, 0xe5, 0xe4, 0xe3, 0xe2, 0xe1, 0xe0,
			},
		},
	}

	for i, c := range cases {
		bhash := bc.NewHash(c.in)
		result := HashToBig(&bhash).Bytes()

		var resArr [32]byte
		copy(resArr[:], result)

		if !reflect.DeepEqual(resArr, c.out) {
			t.Errorf("TestHashToBig test #%d failed:\n\tgot\t%x\n\twant\t%x\n", i, resArr, c.out)
			return
		}
	}
}

func TestCompactToBig(t *testing.T) {
	cases := []struct {
		in  string
		out *big.Int
	}{
		{
			in: `00000000` + //Exponent
				`0` + //Sign
				`0000000000000000000000000000000000000000000000000000000`, //Mantissa
			out: big.NewInt(0),
		},
		{
			in: `00000000` + //Exponent
				`1` + //Sign
				`0000000000000000000000000000000000000000000000000000000`, //Mantissa
			out: big.NewInt(0),
		},
		{
			in: `00000001` + //Exponent
				`0` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(1),
		},
		{
			in: `00000001` + //Exponent
				`1` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(-1),
		},
		{
			in: `00000011` + //Exponent
				`0` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(65536),
		},
		{
			in: `00000011` + //Exponent
				`1` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(-65536),
		},
		{
			in: `00000100` + //Exponent
				`0` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(16777216),
		},
		{
			in: `00000100` + //Exponent
				`1` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(-16777216),
		},
		{
			//btm PowMin test
			// PowMinBits = 2161727821138738707, i.e 0x1e000000000dbe13, as defined
			// in /consensus/general.go
			in: `00011110` + //Exponent
				`0` + //Sign
				`0000000000000000000000000000000000011011011111000010011`, //Mantissa
			out: big.NewInt(0).Lsh(big.NewInt(0x0dbe13), 27*8), //2161727821138738707
		},
	}

	for i, c := range cases {
		compact, _ := strconv.ParseUint(c.in, 2, 64)
		r := CompactToBig(compact)
		if r.Cmp(c.out) != 0 {
			t.Error("TestCompactToBig test #", i, "failed: got", r, "want", c.out)
			return
		}
	}
}

func TestBigToCompact(t *testing.T) {
	// basic tests
	tests := []struct {
		in  int64
		out uint64
	}{
		{0, 0x0000000000000000},
		{-0, 0x0000000000000000},
		{1, 0x0100000000010000},
		{-1, 0x0180000000010000},
		{65536, 0x0300000000010000},
		{-65536, 0x0380000000010000},
		{16777216, 0x0400000000010000},
		{-16777216, 0x0480000000010000},
	}

	for x, test := range tests {
		n := big.NewInt(test.in)
		r := BigToCompact(n)
		if r != test.out {
			t.Errorf("TestBigToCompact test #%d failed: got 0x%016x want 0x%016x\n",
				x, r, test.out)
			return
		}
	}

	// btm PowMin test
	// PowMinBits = 2161727821138738707, i.e 0x1e000000000dbe13, as defined
	// in /consensus/general.go
	n := big.NewInt(0).Lsh(big.NewInt(0x0dbe13), 27*8)
	out := uint64(0x1e000000000dbe13)
	r := BigToCompact(n)
	if r != out {
		t.Errorf("TestBigToCompact test #%d failed: got 0x%016x want 0x%016x\n",
			len(tests), r, out)
		return
	}
}

func TestCalcWorkWithIntStr(t *testing.T) {
	cases := []struct {
		strBits string
		want    *big.Int
	}{
		// Exponent: 0, Sign: 0, Mantissa: 0
		{
			`00000000` + //Exponent
				`0` + //Sign
				`0000000000000000000000000000000000000000000000000000000`, //Mantissa
			big.NewInt(0),
		},
		// Exponent: 0, Sign: 0, Mantissa: 1 (difficultyNum = 0 and difficultyNum.Sign() = 0)
		{
			`00000000` +
				`0` +
				`0000000000000000000000000000000000000000000000000000001`,
			big.NewInt(0),
		},
		// Exponent: 0, Sign: 0, Mantissa: 65536 (difficultyNum = 0 and difficultyNum.Sign() = 0)
		{
			`00000000` +
				`0` +
				`0000000000000000000000000000000000000010000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 0, Sign: 0, Mantissa: 16777216 (difficultyNum = 1 and difficultyNum.Sign() = 1)
		{
			`00000000` +
				`0` +
				`0000000000000000000000000000001000000000000000000000000`,
			new(big.Int).Div(oneLsh256, big.NewInt(2)),
		},
		// Exponent: 0, Sign: 0, Mantissa: 0x007fffffffffffff
		{
			`00000000` +
				`0` +
				`1111111111111111111111111111111111111111111111111111111`,
			big.NewInt(0).Lsh(big.NewInt(0x020000), 208),
		},
		// Exponent: 0, Sign: 1, Mantissa: 0
		{
			`00000000` +
				`1` +
				`0000000000000000000000000000000000000000000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 0, Sign: 1, Mantissa: 1 (difficultyNum = 0 and difficultyNum.Sign() = 0)
		{
			`00000000` +
				`1` +
				`0000000000000000000000000000000000000000000000000000001`,
			big.NewInt(0),
		},
		// Exponent: 0, Sign: 1, Mantissa: 65536 (difficultyNum = 0 and difficultyNum.Sign() = 0)
		{
			`00000000` +
				`1` +
				`0000000000000000000000000000000000000010000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 0, Sign: 0, Mantissa: 16777216 (difficultyNum = -1 and difficultyNum.Sign() = -1)
		{
			`00000000` +
				`1` +
				`0000000000000000000000000000001000000000000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 0, Sign: 1, Mantissa: 0x007fffffffffffff
		{
			`00000000` +
				`1` +
				`1111111111111111111111111111111111111111111111111111111`,
			big.NewInt(0),
		},
		// Exponent: 3, Sign: 0, Mantissa: 0
		{
			`00000011` +
				`0` +
				`0000000000000000000000000000000000000000000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 3, Sign: 0, Mantissa: 1 (difficultyNum = 1 and difficultyNum.Sign() = 1)
		{
			`00000011` +
				`0` +
				`0000000000000000000000000000000000000000000000000000001`,
			new(big.Int).Div(oneLsh256, big.NewInt(2)),
		},
		// Exponent: 3, Sign: 0, Mantissa: 65536 (difficultyNum = 65536 and difficultyNum.Sign() = 1)
		{
			`00000011` +
				`0` +
				`0000000000000000000000000000000000000010000000000000000`,
			new(big.Int).Div(oneLsh256, big.NewInt(65537)),
		},
		// Exponent: 0, Sign: 0, Mantissa: 16777216 (difficultyNum = 16777216 and difficultyNum.Sign() = 1)
		{
			`00000011` +
				`0` +
				`0000000000000000000000000000001000000000000000000000000`,
			new(big.Int).Div(oneLsh256, big.NewInt(16777217)),
		},
		// Exponent: 3, Sign: 0, Mantissa: 0x007fffffffffffff
		{
			`00000011` +
				`0` +
				`1111111111111111111111111111111111111111111111111111111`,
			new(big.Int).Div(oneLsh256, big.NewInt(36028797018963968)),
		},
		// Exponent: 3, Sign: 1, Mantissa: 0
		{
			`00000011` +
				`1` +
				`0000000000000000000000000000000000000000000000000000000`,
			big.NewInt(0),
		},
		//Exponent: 3, Sign: 1, Mantissa: 1 (difficultyNum = -1 and difficultyNum.Sign() = -1)
		{
			`00000011` +
				`1` +
				`0000000000000000000000000000000000000000000000000000001`,
			big.NewInt(0),
		},
		// Exponent: 3, Sign: 1, Mantissa: 65536 (difficultyNum = -65536 and difficultyNum.Sign() = -1)
		{
			`00000011` +
				`1` +
				`0000000000000000000000000000000000000010000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 3, Sign: 1, Mantissa: 16777216 (difficultyNum = -16777216 and difficultyNum.Sign() = -1)
		{
			`00000011` +
				`1` +
				`0000000000000000000000000000001000000000000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 3, Sign: 1, Mantissa: 0x007fffffffffffff
		{
			`00000011` +
				`1` +
				`1111111111111111111111111111111111111111111111111111111`,
			big.NewInt(0),
		},
		// Exponent: 7, Sign: 0, Mantissa: 0
		{
			`00000111` +
				`0` +
				`0000000000000000000000000000000000000000000000000000000`,
			big.NewInt(0),
		},
		//Exponent: 7, Sign: 1, Mantissa: 1 (difficultyNum = 4294967296 and difficultyNum.Sign() = 1)
		{
			`00000111` +
				`0` +
				`0000000000000000000000000000000000000000000000000000001`,
			new(big.Int).Div(oneLsh256, big.NewInt(4294967297)),
		},
		// Exponent: 7, Sign: 0, Mantissa: 65536 (difficultyNum = 4294967296 and difficultyNum.Sign() = 1)
		{
			`00000111` +
				`0` +
				`0000000000000000000000000000000000000010000000000000000`,
			new(big.Int).Div(oneLsh256, big.NewInt(281474976710657)),
		},
		// Exponent: 7, Sign: 0, Mantissa: 16777216 (difficultyNum = 72057594037927936 and difficultyNum.Sign() = 1)
		{
			`00000111` +
				`0` +
				`0000000000000000000000000000001000000000000000000000000`,
			new(big.Int).Div(oneLsh256, big.NewInt(72057594037927937)),
		},
		// Exponent: 7, Sign: 0, Mantissa: 0x007fffffffffffff
		{
			`00000111` +
				`0` +
				`1111111111111111111111111111111111111111111111111111111`,
			new(big.Int).Div(oneLsh256, new(big.Int).Add(big.NewInt(0).Lsh(big.NewInt(36028797018963967), 32), bigOne)),
		},
		// Exponent: 7, Sign: 1, Mantissa: 0
		{
			`00000111` +
				`1` +
				`0000000000000000000000000000000000000000000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 7, Sign: 1, Mantissa: 1 (difficultyNum = -4294967296 and difficultyNum.Sign() = -1)
		{
			`00000111` +
				`1` +
				`0000000000000000000000000000000000000000000000000000001`,
			big.NewInt(0),
		},
		// Exponent: 7, Sign: 1, Mantissa: 65536 (difficultyNum = -72057594037927936 and difficultyNum.Sign() = -1)
		{
			`00000111` +
				`1` +
				`0000000000000000000000000000000000000010000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 7, Sign: 1, Mantissa: 16777216 (difficultyNum = -154742504910672530067423232 and difficultyNum.Sign() = -1)
		{
			`00000111` +
				`1` +
				`0000000000000000000000000000001000000000000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 7, Sign: 1, Mantissa: 0x007fffffffffffff
		{
			`00000111` +
				`1` +
				`1111111111111111111111111111111111111111111111111111111`,
			big.NewInt(0),
		},
		// Exponent: 255, Sign: 0, Mantissa: 1 (difficultyNum.Sign() = 1)
		{
			`11111111` +
				`0` +
				`0000000000000000000000000000000000000000000000000000001`,
			big.NewInt(0),
		},
		// Exponent: 255, Sign: 0, Mantissa: 65536 (difficultyNum.Sign() = 1)
		{
			`11111111` +
				`0` +
				`0000000000000000000000000000000000000010000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 255, Sign: 0, Mantissa: 16777216 (difficultyNum.Sign() = 1)
		{
			`11111111` +
				`0` +
				`0000000000000000000000000000001000000000000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 255, Sign: 0, Mantissa: 0x007fffffffffffff
		{
			`11111111` +
				`0` +
				`1111111111111111111111111111111111111111111111111111111`,
			big.NewInt(0),
		},
		// Exponent: 255, Sign: 1, Mantissa: 1
		{
			`11111111` +
				`1` +
				`0000000000000000000000000000000000000000000000000000001`,
			big.NewInt(0),
		},
		// Exponent: 255, Sign: 1, Mantissa: 65536
		{
			`11111111` +
				`1` +
				`0000000000000000000000000000000000000010000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 255, Sign: 1, Mantissa: 16777216
		{
			`11111111` +
				`1` +
				`0000000000000000000000000000001000000000000000000000000`,
			big.NewInt(0),
		},
		// Exponent: 255, Sign: 1, Mantissa: 0x007fffffffffffff
		{
			`11111111` +
				`1` +
				`1111111111111111111111111111111111111111111111111111111`,
			big.NewInt(0),
		},
	}

	for i, c := range cases {
		bits, err := strconv.ParseUint(c.strBits, 2, 64)
		if err != nil {
			t.Errorf("convert string into uint error: %s\n", err)
			return
		}

		if got := CalcWork(bits); got.Cmp(c.want) != 0 {
			t.Errorf("CalcWork(%d) = %s, want %s\n", i, got, c.want)
			return
		}
	}
}

func TestCalcWork(t *testing.T) {
	testCases := []struct {
		bits uint64
		want *big.Int
	}{
		{
			0,
			big.NewInt(0),
		},
		{
			1,
			big.NewInt(0),
		},
		{
			65535,
			big.NewInt(0),
		},
		{
			16777215,
			big.NewInt(0),
		},
		{
			16777216,
			new(big.Int).Div(oneLsh256, big.NewInt(2)),
		},
		{
			4294967295,
			new(big.Int).Div(oneLsh256, big.NewInt(256)),
		},
		{
			36028797018963967,
			new(big.Int).Div(oneLsh256, big.NewInt(2147483648)),
		},
		{
			36028797018963968,
			big.NewInt(0),
		},
		{
			216172782113783808,
			big.NewInt(0),
		},
		{
			216172782113783809,
			new(big.Int).Div(oneLsh256, big.NewInt(2)),
		},
		{
			216172782130561024,
			new(big.Int).Div(oneLsh256, big.NewInt(16777217)),
		},
		{
			252201579132747775,
			new(big.Int).Div(oneLsh256, big.NewInt(36028797018963968)),
		},
		{
			252201579132747776,
			big.NewInt(0),
		},
		{
			288230376151711744,
			big.NewInt(0),
		},
		{
			288230376151711745,
			new(big.Int).Div(oneLsh256, big.NewInt(257)),
		},
		{
			540431955284459519,
			new(big.Int).Div(oneLsh256, new(big.Int).Add(big.NewInt(0).Lsh(big.NewInt(36028797018963967), 32), bigOne)),
		},
		{
			540431955284459520,
			big.NewInt(0),
		},
		{
			9223372036854775807,
			big.NewInt(0),
		},
		{
			18446744073709551615,
			big.NewInt(0),
		},
	}

	for i, c := range testCases {
		if got := CalcWork(c.bits); got.Cmp(c.want) != 0 {
			t.Errorf("test case with uint64 for CalcWork(%d) = %s, want %s\n", i, got, c.want)
			return
		}
	}
}
