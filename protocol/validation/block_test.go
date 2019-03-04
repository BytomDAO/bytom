package validation

import (
	"math"
	"testing"
	"time"

	"github.com/bytom/consensus"
	"github.com/bytom/mining/tensority"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/protocol/vm/vmutil"
	"github.com/bytom/testutil"
)

func TestCheckBlockTime(t *testing.T) {
	cases := []struct {
		desc       string
		blockTime  uint64
		parentTime []uint64
		err        error
	}{
		{
			blockTime:  1520000001,
			parentTime: []uint64{1520000000},
			err:        nil,
		},
		{
			desc:       "timestamp less than past median time (blocktest#1005)",
			blockTime:  1510000094,
			parentTime: []uint64{1520000000, 1510000099, 1510000098, 1510000097, 1510000096, 1510000095, 1510000094, 1510000093, 1510000092, 1510000091, 1510000090},
			err:        errBadTimestamp,
		},
		{
			desc:       "timestamp greater than max limit (blocktest#1006)",
			blockTime:  9999999999,
			parentTime: []uint64{1520000000},
			err:        errBadTimestamp,
		},
		{
			desc:       "timestamp of the block and the parent block are both greater than max limit (blocktest#1007)",
			blockTime:  uint64(time.Now().Unix()) + consensus.MaxTimeOffsetSeconds + 2,
			parentTime: []uint64{uint64(time.Now().Unix()) + consensus.MaxTimeOffsetSeconds + 1},
			err:        errBadTimestamp,
		},
	}

	parent := &state.BlockNode{Version: 1}
	block := &bc.Block{
		BlockHeader: &bc.BlockHeader{Version: 1},
	}

	for i, c := range cases {
		parent.Timestamp = c.parentTime[0]
		parentSuccessor := parent
		for i := 1; i < len(c.parentTime); i++ {
			parentSuccessor.Parent = &state.BlockNode{Version: 1, Timestamp: c.parentTime[i]}
			parentSuccessor = parentSuccessor.Parent
		}

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
	iniTtensority()

	cases := []struct {
		desc   string
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
			desc: "the difficulty of the block is not equals to the next difficulty of parent block (blocktest#1008)",
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
			desc: "the prev block hash not equals to the hash of parent (blocktest#1004)",
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
			desc: "check work proof fail (blocktest#1011)",
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
		{
			desc: "version greater than 1 (blocktest#1001)",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version: 2,
				},
			},
			parent: &state.BlockNode{
				Version: 1,
			},
			err: errVersionRegression,
		},
		{
			desc: "version equals 0 (blocktest#1002)",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version: 0,
				},
			},
			parent: &state.BlockNode{
				Version: 1,
			},
			err: errVersionRegression,
		},
		{
			desc: "version equals max uint64 (blocktest#1003)",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version: math.MaxUint64,
				},
			},
			parent: &state.BlockNode{
				Version: 1,
			},
			err: errVersionRegression,
		},
	}

	for i, c := range cases {
		if err := ValidateBlockHeader(c.block, c.parent); rootErr(err) != c.err {
			t.Errorf("case %d (%s) got error %s, want %s", i, c.desc, err, c.err)
		}
	}
}

// TestValidateBlock test the ValidateBlock function
func TestValidateBlock(t *testing.T) {
	iniTtensority()

	cp, _ := vmutil.DefaultCoinbaseProgram()
	cases := []struct {
		desc   string
		block  *bc.Block
		parent *state.BlockNode
		err    error
	}{
		{
			desc: "The calculated transaction merkel root hash is not equals to the hash of the block header (blocktest#1009)",
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
			desc: "The calculated transaction status merkel root hash is not equals to the hash of the block header (blocktest#1009)",
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
		{
			desc: "the coinbase amount is less than the real coinbase amount (txtest#1014)",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:         1,
					Height:          1,
					Timestamp:       1523352601,
					PreviousBlockId: &bc.Hash{V0: 0},
					Bits:            2305843009214532812,
				},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs:        []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 41250000000, cp)},
					}),
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp)},
						Outputs:        []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 90000000, cp)},
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
			err: ErrWrongCoinbaseTransaction,
		},
	}

	for i, c := range cases {
		err := ValidateBlock(c.block, c.parent)
		if rootErr(err) != c.err {
			t.Errorf("case #%d (%s) got error %s, want %s", i, c.desc, err, c.err)
		}
	}
}

// TestGasOverBlockLimit check if the gas of the block has the max limit (blocktest#1012)
func TestGasOverBlockLimit(t *testing.T) {
	iniTtensority()

	cp, _ := vmutil.DefaultCoinbaseProgram()
	parent := &state.BlockNode{
		Version:   1,
		Height:    0,
		Timestamp: 1523352600,
		Hash:      bc.Hash{V0: 0},
		Seed:      &bc.Hash{V1: 1},
		Bits:      2305843009214532812,
	}
	block := &bc.Block{
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
	}

	for i := 0; i < 100; i++ {
		block.Transactions = append(block.Transactions, types.MapTx(&types.TxData{
			Version:        1,
			SerializedSize: 100000,
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 10000000000, 0, cp),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 9000000000, cp),
			},
		}))
	}

	if err := ValidateBlock(block, parent); err != errOverBlockLimit {
		t.Errorf("got error %s, want %s", err, errOverBlockLimit)
	}
}

// TestSetTransactionStatus verify the transaction status is set correctly (blocktest#1010)
func TestSetTransactionStatus(t *testing.T) {
	iniTtensority()

	cp, _ := vmutil.DefaultCoinbaseProgram()
	parent := &state.BlockNode{
		Version:   1,
		Height:    0,
		Timestamp: 1523352600,
		Hash:      bc.Hash{V0: 0},
		Seed:      &bc.Hash{V1: 1},
		Bits:      2305843009214532812,
	}
	block := &bc.Block{
		ID: bc.Hash{V0: 1},
		BlockHeader: &bc.BlockHeader{
			Version:               1,
			Height:                1,
			Timestamp:             1523352601,
			PreviousBlockId:       &bc.Hash{V0: 0},
			Bits:                  2305843009214532812,
			TransactionsRoot:      &bc.Hash{V0: 3413931728524254295, V1: 300490676707850231, V2: 1886132055969225110, V3: 10216139531293906088},
			TransactionStatusHash: &bc.Hash{V0: 8682965660674182538, V1: 8424137560837623409, V2: 6979974817894224946, V3: 4673809519342015041},
		},
		Transactions: []*bc.Tx{
			types.MapTx(&types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
				Outputs:        []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 41449998224, cp)},
			}),
			types.MapTx(&types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
					types.NewSpendInput([][]byte{}, *newHash(8), bc.AssetID{V0: 1}, 1000, 0, []byte{byte(vm.OP_FALSE)}),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 888, cp),
					types.NewTxOutput(bc.AssetID{V0: 1}, 1000, cp),
				},
			}),
			types.MapTx(&types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 888, cp),
				},
			}),
		},
	}

	if err := ValidateBlock(block, parent); err != nil {
		t.Fatal(err)
	}

	expectTxStatuses := []bool{false, true, false}
	txStatuses := block.GetTransactionStatus().VerifyStatus
	if len(expectTxStatuses) != len(txStatuses) {
		t.Error("the size of expect tx status is not equals to size of got tx status")
	}

	for i, status := range txStatuses {
		if expectTxStatuses[i] != status.StatusFail {
			t.Errorf("got tx status: %v, expect tx status: %v\n", status.StatusFail, expectTxStatuses[i])
		}
	}
}

func iniTtensority() {
	// add (hash, seed) --> (tensority hash) to the  tensority cache for avoid
	// real matrix calculate cost.
	tensority.AIHash.AddCache(&bc.Hash{V0: 0}, &bc.Hash{}, testutil.MaxHash)
	tensority.AIHash.AddCache(&bc.Hash{V0: 1}, &bc.Hash{}, testutil.MinHash)
	tensority.AIHash.AddCache(&bc.Hash{V0: 1}, consensus.InitialSeed, testutil.MinHash)
}
