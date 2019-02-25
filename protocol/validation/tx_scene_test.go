package validation

import (
	"math"
	"testing"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/protocol/vm/vmutil"
)

func TestValidateTx(t *testing.T) {
	cp, _ := vmutil.DefaultCoinbaseProgram()
	prog, err := vm.Assemble("ADD 5 NUMEQUAL")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		desc     string
		txData   *types.TxData
		gasValid bool
		err      error
	}{
		{
			desc: "transaction has no output",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
				},
			},
			gasValid: false,
			err:      ErrEmptyResults,
		},
		{
			desc: "sum of the output btm asset greater than input btm asset",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 100000001, cp),
				},
			},
			gasValid: false,
			err:      ErrGasCalculate,
		},
		{
			desc: "sum of the input btm asset is overflow",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, math.MaxInt64, 0, cp),
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 1, 0, cp),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 100000000, cp),
				},
			},
			gasValid: false,
			err:      ErrOverflow,
		},
		{
			desc: "issuance input has no corresponding output",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewIssuanceInput([]byte{3}, 10, []byte{1}, [][]byte{[]byte{2}, []byte{3}}, []byte{2}),
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 50000000, cp),
				},
			},
			gasValid: false,
			err:      ErrUnbalanced,
		},
		{
			desc: "issuance asset A, but output asset B",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewIssuanceInput([]byte{3}, 10, prog, [][]byte{[]byte{2}, []byte{3}}, []byte{2}),
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(bc.AssetID{V0: 0, V1: 1, V2: 2, V3: 3}, 10, cp),
					types.NewTxOutput(*consensus.BTMAssetID, 50000000, cp),
				},
			},
			gasValid: false,
			err:      ErrNoSource,
		},
		{
			desc: "issuance transaction has no gas input",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewIssuanceInput([]byte{3}, 10, prog, [][]byte{[]byte{2}, []byte{3}}, []byte{2}),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(bc.AssetID{V0: 2596100136807737023, V1: 13466434218884500603, V2: 916280593609723488, V3: 439048993277404648}, 10, cp),
				},
			},
			gasValid: true, // TODO It's a bug, need hard fork solution
			err:      vm.ErrRunLimitExceeded,
		},
		{
			desc: "input using the same utxo",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 180000000, cp),
				},
			},
			gasValid: true,
			err:      ErrMismatchedPosition,
		},
		{
			desc: "output with over range amount but sum in equal",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 18446744073609551616, cp),
					types.NewTxOutput(*consensus.BTMAssetID, 18446744073609551616, cp),
					types.NewTxOutput(*consensus.BTMAssetID, 290000000, cp),
				},
			},
			gasValid: false,
			err:      ErrOverflow,
		},
		{
			desc: "sum of output greater than sum of input",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 10, 0, cp),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 20, cp),
				},
			},
			gasValid: false,
			err:      ErrGasCalculate,
		},
	}

	for i, c := range cases {
		gasStatus, err := ValidateTx(types.MapTx(c.txData), mockBlock())
		if rootErr(err) != c.err {
			t.Errorf("case #%d (%s) got error %s, want %s; validationState is:\n", i, c.desc, err, c.err)
		}
		if c.gasValid != gasStatus.GasValid {
			t.Errorf("#%d got GasValid %t, want %t", i, gasStatus.GasValid, c.gasValid)
		}
	}
}
