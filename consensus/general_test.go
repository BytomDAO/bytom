package consensus

import "testing"

func TestSubsidy(t *testing.T) {
	cases := []struct {
		subsidy uint64
		height  uint64
	}{
		{
			subsidy: baseSubsidy,
			height:  1,
		},
		{
			subsidy: baseSubsidy,
			height:  subsidyReductionInterval - 1,
		},
		{
			subsidy: baseSubsidy / 2,
			height:  subsidyReductionInterval,
		},
		{
			subsidy: baseSubsidy / 2,
			height:  subsidyReductionInterval + 1,
		},
		{
			subsidy: baseSubsidy / 1024,
			height:  subsidyReductionInterval * 10,
		},
	}

	for _, c := range cases {
		subsidy := BlockSubsidy(c.height)
		if subsidy != c.subsidy {
			t.Errorf("got subsidy %d, want %d", subsidy, c.subsidy)
		}
	}
}
