package validation

import (
	"testing"

	"github.com/bytom/consensus"
	"github.com/bytom/mining/tensority"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
	"github.com/bytom/protocol/vm/vmutil"
	"github.com/bytom/testutil"
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

	parent := &state.BlockNode{Version: 1}
	block := &bc.Block{
		BlockHeader: &bc.BlockHeader{Version: 1},
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
			err:    ErrWrongCoinbaseTransaction,
		},
		{
			txs:    []*types.Tx{},
			amount: 5000,
			err:    ErrWrongCoinbaseTransaction,
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

func TestValidateBlockHeader(t *testing.T) {
	// add (hash, seed) --> (tensority hash) to the  tensority cache for avoid
	// real matrix calculate cost.
	tensority.AIHash.AddCache(&bc.Hash{V0: 0}, &bc.Hash{}, testutil.MaxHash)
	tensority.AIHash.AddCache(&bc.Hash{V0: 1}, &bc.Hash{}, testutil.MinHash)
	tensority.AIHash.AddCache(&bc.Hash{V0: 1}, consensus.InitialSeed, testutil.MinHash)

	cases := []struct {
		block  *bc.Block
		parent *state.BlockNode
		err    error
	}{
		{
			block: &bc.Block{BlockHeader: &bc.BlockHeader{
				Version: 2,
			}},
			parent: &state.BlockNode{
				Version: 1,
			},
			err: errVersionRegression,
		},
		{
			block: &bc.Block{BlockHeader: &bc.BlockHeader{
				Version: 1,
				Height:  20,
			}},
			parent: &state.BlockNode{
				Version: 1,
				Height:  18,
			},
			err: errMisorderedBlockHeight,
		},
		{
			block: &bc.Block{BlockHeader: &bc.BlockHeader{
				Version: 1,
				Height:  20,
				Bits:    0,
			}},
			parent: &state.BlockNode{
				Version: 1,
				Height:  19,
				Bits:    2305843009214532812,
			},
			err: errBadBits,
		},
		{
			block: &bc.Block{BlockHeader: &bc.BlockHeader{
				Version:         1,
				Height:          20,
				PreviousBlockId: &bc.Hash{V0: 18},
			}},
			parent: &state.BlockNode{
				Version: 1,
				Height:  19,
				Hash:    bc.Hash{V0: 19},
			},
			err: errMismatchedBlock,
		},
		{
			block: &bc.Block{
				ID: bc.Hash{V0: 0},
				BlockHeader: &bc.BlockHeader{
					Version:         1,
					Height:          1,
					Timestamp:       1523352601,
					PreviousBlockId: &bc.Hash{V0: 0},
					Bits:            2305843009214532812,
				},
			},
			parent: &state.BlockNode{
				Version:   1,
				Height:    0,
				Timestamp: 1523352600,
				Hash:      bc.Hash{V0: 0},
				Seed:      &bc.Hash{V1: 1},
				Bits:      2305843009214532812,
			},
			err: errWorkProof,
		},
		{
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:         1,
					Height:          1,
					Timestamp:       1523352601,
					PreviousBlockId: &bc.Hash{V0: 0},
					Bits:            2305843009214532812,
				},
			},
			parent: &state.BlockNode{
				Version:   1,
				Height:    0,
				Timestamp: 1523352600,
				Hash:      bc.Hash{V0: 0},
				Seed:      &bc.Hash{V1: 1},
				Bits:      2305843009214532812,
			},
			err: nil,
		},
	}

	for i, c := range cases {
		if err := ValidateBlockHeader(c.block, c.parent); rootErr(err) != c.err {
			t.Errorf("case %d got error %s, want %s", i, err, c.err)
		}
	}
}

func TestValidateMerkleRoot(t *testing.T) {
	// add (hash, seed) --> (tensority hash) to the  tensority cache for avoid
	// real matrix calculate cost.
	tensority.AIHash.AddCache(&bc.Hash{V0: 0}, &bc.Hash{}, testutil.MaxHash)
	tensority.AIHash.AddCache(&bc.Hash{V0: 1}, &bc.Hash{}, testutil.MinHash)
	tensority.AIHash.AddCache(&bc.Hash{V0: 1}, consensus.InitialSeed, testutil.MinHash)

	cp, _ := vmutil.DefaultCoinbaseProgram()
	cases := []struct {
		desc   string
		block  *bc.Block
		parent *state.BlockNode
		err    error
	}{
		{
			desc: "The calculated transaction merkel root hash is not equals to the hash of the block header",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:          1,
					Height:           1,
					Timestamp:        1523352601,
					PreviousBlockId:  &bc.Hash{V0: 0},
					Bits:             2305843009214532812,
					TransactionsRoot: &bc.Hash{V0: 1},
				},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs:        []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 41250000000, cp)},
					}),
				},
			},
			parent: &state.BlockNode{
				Version:   1,
				Height:    0,
				Timestamp: 1523352600,
				Hash:      bc.Hash{V0: 0},
				Seed:      &bc.Hash{V1: 1},
				Bits:      2305843009214532812,
			},
			err: errMismatchedMerkleRoot,
		},
		{
			desc: "The calculated transaction status merkel root hash is not equals to the hash of the block header",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:               1,
					Height:                1,
					Timestamp:             1523352601,
					PreviousBlockId:       &bc.Hash{V0: 0},
					Bits:                  2305843009214532812,
					TransactionsRoot:      &bc.Hash{V0: 6294987741126419124, V1: 12520373106916389157, V2: 5040806596198303681, V3: 1151748423853876189},
					TransactionStatusHash: &bc.Hash{V0: 1},
				},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs:        []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 41250000000, cp)},
					}),
				},
			},
			parent: &state.BlockNode{
				Version:   1,
				Height:    0,
				Timestamp: 1523352600,
				Hash:      bc.Hash{V0: 0},
				Seed:      &bc.Hash{V1: 1},
				Bits:      2305843009214532812,
			},
			err: errMismatchedMerkleRoot,
		},
	}

	for i, c := range cases {
		err := ValidateBlock(c.block, c.parent)
		if rootErr(err) != c.err {
			t.Errorf("case #%d (%s) got error %s, want %s", i, c.desc, err, c.err)
		}
	}
}
