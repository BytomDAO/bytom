package mining

import "testing"

func TestCreateCoinbaseTx(t *testing.T) {
	reductionInterval := uint64(840000)
	baseSubsidy := uint64(41250000000)
	cases := []struct {
		height  uint64
		txFee   uint64
		subsidy uint64
	}{
		{
			height:  reductionInterval - 1,
			txFee:   100000000,
			subsidy: baseSubsidy + 100000000,
		},
		{
			height:  reductionInterval,
			txFee:   2000000000,
			subsidy: baseSubsidy/2 + 2000000000,
		},
		{
			height:  reductionInterval + 1,
			txFee:   0,
			subsidy: baseSubsidy / 2,
		},
		{
			height:  reductionInterval * 2,
			txFee:   100000000,
			subsidy: baseSubsidy/4 + 100000000,
		},
	}

	for _, c := range cases {
		coinbaseTx, err := createCoinbaseTx(nil, c.txFee, c.height)
		if err != nil {
			t.Fatal(err)
		}

		outputAmount := coinbaseTx.Outputs[0].OutputCommitment.Amount
		if outputAmount != c.subsidy {
			t.Fatalf("coinbase tx reward dismatch, expected: %d, have: %d", c.subsidy, outputAmount)
		}
	}
}
