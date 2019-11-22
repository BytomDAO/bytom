// +build functional

package test

import (
	"os"
	"testing"
	"time"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/vm"
	dbm "github.com/bytom/bytom/database/leveldb"
)

func TestBlockHeader(t *testing.T) {
	db := dbm.NewDB("block_test_db", "leveldb", "block_test_db")
	defer os.RemoveAll("block_test_db")
	chain, _, _, _ := MockChain(db)
	genesisHeader := chain.BestBlockHeader()
	if err := AppendBlocks(chain, 1); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		desc       string
		version    func() uint64
		prevHeight func() uint64
		timestamp  func() uint64
		prevHash   func() *bc.Hash
		bits       func() uint64
		solve      bool
		valid      bool
	}{
		{
			desc:       "block version is 1",
			version:    func() uint64 { return 1 },
			prevHeight: chain.BestBlockHeight,
			timestamp:  func() uint64 { return chain.BestBlockHeader().Timestamp + 1 },
			prevHash:   chain.BestBlockHash,
			bits:       func() uint64 { return chain.BestBlockHeader().Bits },
			solve:      true,
			valid:      true,
		},
		{
			desc:       "invalid block, misorder block height",
			version:    func() uint64 { return chain.BestBlockHeader().Version },
			prevHeight: func() uint64 { return chain.BestBlockHeight() + 1 },
			timestamp:  func() uint64 { return chain.BestBlockHeader().Timestamp + 1 },
			prevHash:   chain.BestBlockHash,
			bits:       func() uint64 { return chain.BestBlockHeader().Bits },
			solve:      true,
			valid:      false,
		},
		{
			desc:       "invalid prev hash, prev hash dismatch",
			version:    func() uint64 { return chain.BestBlockHeader().Version },
			prevHeight: chain.BestBlockHeight,
			timestamp:  func() uint64 { return chain.BestBlockHeader().Timestamp + 1 },
			prevHash:   func() *bc.Hash { hash := genesisHeader.Hash(); return &hash },
			bits:       func() uint64 { return chain.BestBlockHeader().Bits },
			solve:      true,
			valid:      false,
		},
		{
			desc:       "invalid bits",
			version:    func() uint64 { return chain.BestBlockHeader().Version },
			prevHeight: chain.BestBlockHeight,
			timestamp:  func() uint64 { return chain.BestBlockHeader().Timestamp + 1 },
			prevHash:   chain.BestBlockHash,
			bits:       func() uint64 { return chain.BestBlockHeader().Bits + 100 },
			solve:      true,
			valid:      false,
		},
		{
			desc:       "invalid timestamp, greater than MaxTimeOffsetSeconds from system time",
			version:    func() uint64 { return chain.BestBlockHeader().Version },
			prevHeight: chain.BestBlockHeight,
			timestamp:  func() uint64 { return uint64(time.Now().Unix()) + consensus.MaxTimeOffsetSeconds + 60 },
			prevHash:   chain.BestBlockHash,
			bits:       func() uint64 { return chain.BestBlockHeader().Bits },
			solve:      true,
			valid:      false,
		},
		{
			desc:       "valid timestamp, greater than last block",
			version:    func() uint64 { return chain.BestBlockHeader().Version },
			prevHeight: chain.BestBlockHeight,
			timestamp:  func() uint64 { return chain.BestBlockHeader().Timestamp + 3 },
			prevHash:   chain.BestBlockHash,
			bits:       func() uint64 { return chain.BestBlockHeader().Bits },
			solve:      true,
			valid:      true,
		},
		{
			desc:       "valid timestamp, less than last block, but greater than median",
			version:    func() uint64 { return chain.BestBlockHeader().Version },
			prevHeight: chain.BestBlockHeight,
			timestamp:  func() uint64 { return chain.BestBlockHeader().Timestamp - 1 },
			prevHash:   chain.BestBlockHash,
			bits:       func() uint64 { return chain.BestBlockHeader().Bits },
			solve:      true,
			valid:      true,
		},
		{
			desc:       "invalid timestamp, less than median",
			version:    func() uint64 { return chain.BestBlockHeader().Version },
			prevHeight: chain.BestBlockHeight,
			timestamp:  func() uint64 { return genesisHeader.Timestamp },
			prevHash:   chain.BestBlockHash,
			bits:       func() uint64 { return chain.BestBlockHeader().Bits },
			solve:      true,
			valid:      false,
		},
	}

	for _, c := range cases {
		block, err := NewBlock(chain, nil, []byte{byte(vm.OP_TRUE)})
		if err != nil {
			t.Fatal(err)
		}

		block.Version = c.version()
		block.Height = c.prevHeight() + 1
		block.Timestamp = c.timestamp()
		block.PreviousBlockHash = *c.prevHash()
		block.Bits = c.bits()
		seed, err := chain.CalcNextSeed(&block.PreviousBlockHash)
		if err != nil && c.valid {
			t.Fatal(err)
		}

		if c.solve {
			Solve(seed, block)
		}
		_, err = chain.ProcessBlock(block)
		result := err == nil
		if result != c.valid {
			t.Fatalf("%s test failed, expected: %t, have: %t, err: %s", c.desc, c.valid, result, err)
		}
	}
}

func TestMaxBlockGas(t *testing.T) {
	chainDB := dbm.NewDB("test_block_db", "leveldb", "test_block_db")
	defer os.RemoveAll("test_block_db")
	chain, _, _, err := MockChain(chainDB)
	if err != nil {
		t.Fatal(err)
	}

	if err := AppendBlocks(chain, 7); err != nil {
		t.Fatal(err)
	}

	block, err := chain.GetBlockByHeight(1)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := CreateTxFromTx(block.Transactions[0], 0, 600000000000, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	outputAmount := uint64(600000000000)
	txs := []*types.Tx{tx}
	for i := 1; i < 50000; i++ {
		outputAmount -= 10000000
		tx, err := CreateTxFromTx(txs[i-1], 0, outputAmount, []byte{byte(vm.OP_TRUE)})
		if err != nil {
			t.Fatal(err)
		}
		txs = append(txs, tx)
	}

	block, err = NewBlock(chain, txs, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	if err := SolveAndUpdate(chain, block); err == nil {
		t.Fatalf("test max block gas failed")
	}
}
