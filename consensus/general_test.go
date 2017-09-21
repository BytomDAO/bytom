package consensus

import (
	"fmt"
	"math/big"
	"testing"
)

func TestCalcNextRequiredDifficulty(t *testing.T) {
	//fmt.Println(CalcNextRequiredDifficulty())
	x := big.NewInt(123)
	y, _ := x.SetString("94847123945178081620347972471576132812524935594538618173381454864040345", 10)
	fmt.Println(BigToCompact(y))
}

/*func TestSubsidy(t *testing.T) {
	cases := []struct {
		bh      *BlockHeader
		subsidy uint64
	}{
		{
			bh: &BlockHeader{
				Height: 1,
			},
			subsidy: 624000000000,
		},
		{
			bh: &BlockHeader{
				Height: 560640,
			},
			subsidy: 312000000000,
		},
	}

	for _, c := range cases {
		subsidy := c.bh.BlockSubsidy()

		if subsidy != c.subsidy {
			t.Errorf("got subsidy %s, want %s", subsidy, c.subsidy)
		}
	}
}*/
