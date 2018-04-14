package mining

import (
	"fmt"
	"testing"
)

func TestCreateCoinbaseTx(t *testing.T) {
	reductionInterval := uint64(560640)
	baseReward := uint64(41250000000)
	cases := []struct {
		height uint64
		txFee  uint64
		reward uint64
	}{
		{
			height: reductionInterval - 1,
			txFee:  0,
			reward: baseReward,
		},
		{
			height: reductionInterval,
			txFee:  0,
			reward: baseReward / 2,
		},
		{
			height: reductionInterval + 1,
			txFee:  0,
			reward: baseReward / 2,
		},
		{
			height: reductionInterval * 2,
			txFee:  100000000,
			reward: baseReward/4 + 100000000,
		},
		{
			height: reductionInterval * 10,
			txFee:  0,
			reward: baseReward / 1024,
		},
	}

	for _, c := range cases {
		coinbaseTx, err := createCoinbaseTx(nil, c.txFee, c.height)
		if err != nil {
			t.Fatal(err)
		}

		outputAmount := coinbaseTx.Outputs[0].OutputCommitment.Amount
		fmt.Println(outputAmount)
		if outputAmount != c.reward {
			t.Fatalf("coinbase tx reward dismatch, expected: %d, have: %d", c.reward, outputAmount)
		}
	}
}
