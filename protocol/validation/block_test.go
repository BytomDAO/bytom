package validation

import (
	"encoding/hex"
	"math"
	"testing"
	"time"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

func TestCheckBlockTime(t *testing.T) {
	cases := []struct {
		desc       string
		blockTime  uint64
		parentTime []uint64
		err        error
	}{
		{
			blockTime:  1520006000,
			parentTime: []uint64{1520000000},
			err:        nil,
		},
		{
			desc:       "timestamp less than past median time",
			blockTime:  1520006000,
			parentTime: []uint64{1520000000, 1520000500, 1520001000, 1520001500, 1520002000, 1520002500, 1520003000, 1520003500, 1520004000, 1520004500, 1520005000},
			err:        nil,
		},
		{
			desc:       "timestamp greater than max limit",
			blockTime:  99999999990000,
			parentTime: []uint64{15200000000000},
			err:        errBadTimestamp,
		},
		{
			desc:       "timestamp of the block and the parent block are both greater than max limit",
			blockTime:  uint64(time.Now().UnixNano()/int64(time.Millisecond)) + consensus.ActiveNetParams.MaxTimeOffsetMs + 2000,
			parentTime: []uint64{uint64(time.Now().UnixNano()/int64(time.Millisecond)) + consensus.ActiveNetParams.MaxTimeOffsetMs + 1000},
			err:        errBadTimestamp,
		},
	}

	parent := &types.BlockHeader{Version: 1}
	block := &types.BlockHeader{Version: 1}
	for i, c := range cases {
		parent.Timestamp = c.parentTime[0]
		parentSuccessor := parent
		for i := 1; i < len(c.parentTime); i++ {
			Previous := &types.BlockHeader{Version: 1, Timestamp: c.parentTime[i]}
			parentSuccessor.PreviousBlockHash = Previous.Hash()
			parentSuccessor = Previous
		}

		block.Timestamp = c.blockTime
		if err := checkBlockTime(block, parent); rootErr(err) != c.err {
			t.Errorf("case %d got error %s, want %s", i, err, c.err)
		}
	}
}

func TestCheckCoinbaseAmount(t *testing.T) {
	cases := []struct {
		block      *types.Block
		checkpoint *state.Checkpoint
		err        error
	}{
		{
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: 0},
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs: []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs: []*types.TxOutput{
							types.NewOriginalTxOutput(*consensus.BTMAssetID, 0, []byte("controlProgram"), nil),
						},
					}),
				},
			},
			checkpoint: &state.Checkpoint{
				Rewards: map[string]uint64{hex.EncodeToString([]byte("controlProgram")): 5000},
			},
			err: nil,
		},
		{
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: consensus.ActiveNetParams.BlocksOfEpoch + 1},
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs: []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs: []*types.TxOutput{
							types.NewOriginalTxOutput(*consensus.BTMAssetID, 5000, []byte("controlProgram"), nil),
						},
					}),
				},
			},
			checkpoint: &state.Checkpoint{
				Rewards: map[string]uint64{hex.EncodeToString([]byte("controlProgram")): 5000},
			},
			err: nil,
		},
		{
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: consensus.ActiveNetParams.BlocksOfEpoch + 1},
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs: []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs: []*types.TxOutput{
							types.NewOriginalTxOutput(*consensus.BTMAssetID, 0, []byte("controlProgramX1"), nil),
							types.NewOriginalTxOutput(*consensus.BTMAssetID, 5000, []byte("controlProgram"), nil),
						},
					}),
				},
			},
			checkpoint: &state.Checkpoint{
				Rewards: map[string]uint64{hex.EncodeToString([]byte("controlProgram")): 5000},
			},
			err: nil,
		},
		{
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: consensus.ActiveNetParams.BlocksOfEpoch + 1},
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs: []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs: []*types.TxOutput{
							types.NewOriginalTxOutput(*consensus.BTMAssetID, 1000, []byte("controlProgram"), nil),
							types.NewOriginalTxOutput(*consensus.BTMAssetID, 5000, []byte("controlProgram"), nil),
						},
					}),
				},
			},
			checkpoint: &state.Checkpoint{
				Rewards: map[string]uint64{hex.EncodeToString([]byte("controlProgram")): 5000},
			},
			err: ErrWrongCoinbaseTransaction,
		},
		{
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: consensus.ActiveNetParams.BlocksOfEpoch + 1},
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs: []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs: []*types.TxOutput{
							types.NewOriginalTxOutput(*consensus.BTMAssetID, 5000, []byte("controlProgram1"), nil),
							types.NewOriginalTxOutput(*consensus.BTMAssetID, 5000, []byte("controlProgram2"), nil),
						},
					}),
				},
			},
			checkpoint: &state.Checkpoint{
				Rewards: map[string]uint64{
					hex.EncodeToString([]byte("controlProgram1")): 5000,
					hex.EncodeToString([]byte("controlProgram2")): 5000},
			},
			err: nil,
		},
		{
			block: &types.Block{
				BlockHeader:  types.BlockHeader{},
				Transactions: []*types.Tx{},
			},
			checkpoint: &state.Checkpoint{
				Rewards: map[string]uint64{"controlProgram": 5000},
			},
			err: ErrWrongCoinbaseTransaction,
		},
	}

	for i, c := range cases {
		if err := checkCoinbaseAmount(c.block, c.checkpoint); rootErr(err) != c.err {
			t.Errorf("case %d got error %v, want %v", i, err, c.err)
		}
	}
}

func TestValidateBlockHeader(t *testing.T) {
	cases := []struct {
		desc   string
		bh     *types.BlockHeader
		parent *types.BlockHeader
		err    error
	}{
		{
			bh: &types.BlockHeader{
				Version: 2,
			},
			parent: &types.BlockHeader{
				Version: 1,
			},
			err: errVersionRegression,
		},
		{
			bh: &types.BlockHeader{
				Version: 1,
				Height:  20,
			},
			parent: &types.BlockHeader{
				Version: 1,
				Height:  18,
			},
			err: errMisorderedBlockHeight,
		},
		{
			desc: "the prev block hash not equals to the hash of parent (blocktest#1004)",
			bh: &types.BlockHeader{
				Version:           1,
				Height:            20,
				Timestamp:         1523358600,
				PreviousBlockHash: bc.Hash{V0: 0},
			},
			parent: &types.BlockHeader{
				Version: 1,
				Height:  19,
			},
			err: errMismatchedBlock,
		},
		{
			desc: "version greater than 1 (blocktest#1001)",
			bh: &types.BlockHeader{
				Version: 2,
			},
			parent: &types.BlockHeader{
				Version: 1,
			},
			err: errVersionRegression,
		},
		{
			desc: "version equals 0 (blocktest#1002)",
			bh: &types.BlockHeader{
				Version: 0,
			},
			parent: &types.BlockHeader{
				Version: 1,
			},
			err: errVersionRegression,
		},
		{
			desc: "version equals max uint64 (blocktest#1003)",
			bh: &types.BlockHeader{
				Version: math.MaxUint64,
			},
			parent: &types.BlockHeader{
				Version: 1,
			},
			err: errVersionRegression,
		},
	}

	for i, c := range cases {
		if err := ValidateBlockHeader(c.bh, c.parent, nil); rootErr(err) != c.err {
			t.Errorf("case %d (%s) got error %s, want %s", i, c.desc, err, c.err)
		}
	}
}
