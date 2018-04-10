package validation

import (
	"testing"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
)

func TestCheckBlockTime(t *testing.T) {
	cases := []struct {
		blockTime  uint64
		parentTime uint64
		err        error
	}{
		{
			blockTime:  1520000001,
			parentTime: 1520000000,
			err:        nil,
		},
		{
			blockTime:  1510000000,
			parentTime: 1520000000,
			err:        errBadTimestamp,
		},
		{
			blockTime:  9999999999,
			parentTime: 1520000000,
			err:        errBadTimestamp,
		},
	}

	parent := &state.BlockNode{}
	block := &bc.Block{
		BlockHeader: &bc.BlockHeader{},
	}

	for i, c := range cases {
		parent.Timestamp = c.parentTime
		block.Timestamp = c.blockTime
		if err := checkBlockTime(block, parent); rootErr(err) != c.err {
			t.Errorf("case %d got error %s, want %s", i, err, c.err)
		}
	}
}

func TestCheckCoinbaseAmount(t *testing.T) {
	cases := []struct {
		txs    []*types.Tx
		amount uint64
		err    error
	}{
		{
			txs: []*types.Tx{
				types.NewTx(types.TxData{
					Inputs:  []*types.TxInput{types.NewCoinbaseInput(nil)},
					Outputs: []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 5000, nil)},
				}),
			},
			amount: 5000,
			err:    nil,
		},
		{
			txs: []*types.Tx{
				types.NewTx(types.TxData{
					Inputs:  []*types.TxInput{types.NewCoinbaseInput(nil)},
					Outputs: []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 5000, nil)},
				}),
			},
			amount: 6000,
			err:    errWrongCoinbaseTransaction,
		},
		{
			txs:    []*types.Tx{},
			amount: 5000,
			err:    errWrongCoinbaseTransaction,
		},
	}

	block := new(types.Block)
	for i, c := range cases {
		block.Transactions = c.txs
		if err := checkCoinbaseAmount(types.MapBlock(block), c.amount); rootErr(err) != c.err {
			t.Errorf("case %d got error %s, want %s", i, err, c.err)
		}
	}
}
