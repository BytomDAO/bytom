package consensus

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
