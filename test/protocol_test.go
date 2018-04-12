// +build functional

package test

import (
	"os"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
)

// case1:           |------c1(height=7)
// --------(height=5)
//                  |------------c2(height=9)
func TestForkCase1(t *testing.T) {
	c1, err := declChain("chain1", nil, 0, 7)
	defer os.RemoveAll("chain1")
	if err != nil {
		t.Fatal(err)
	}

	c2, err := declChain("chain2", c1, 5, 9)
	defer os.RemoveAll("chain2")
	if err != nil {
		t.Fatal(err)
	}

	bestBlockHash := c2.BestBlockHash()
	if err := merge(c1, c2); err != nil {
		t.Fatal(err)
	}

	if *c1.BestBlockHash() != *bestBlockHash || *c2.BestBlockHash() != *bestBlockHash {
		t.Fatalf("test fork case1 failed")
	}

	if !c1.InMainChain(*bestBlockHash) || !c2.InMainChain(*bestBlockHash) {
		t.Fatalf("best block is not in main chain")
	}
}

// case2:            |----c1(height=6)
// ---------(height 5)
//                   |----c2(height=6)
func TestForkCase2(t *testing.T) {
	c1, err := declChain("chain1", nil, 0, 6)
	defer os.RemoveAll("chain1")
	if err != nil {
		t.Fatal(err)
	}

	c2, err := declChain("chain2", c1, 5, 6)
	defer os.RemoveAll("chain2")
	if err != nil {
		t.Fatal(err)
	}

	c1BestBlockHash := c1.BestBlockHash()
	c2BestBlockHash := c2.BestBlockHash()
	if err := merge(c1, c2); err != nil {
		t.Fatal(err)
	}

	if *c1.BestBlockHash() != *c1BestBlockHash || *c2.BestBlockHash() != *c2BestBlockHash {
		t.Fatalf("test fork case2 failed")
	}

	if !c1.InMainChain(*c1BestBlockHash) || !c2.InMainChain(*c2BestBlockHash) {
		t.Fatalf("best block is not in main chain")
	}
}

func TestBlockSync(t *testing.T) {
	c1, err := declChain("chain1", nil, 0, 5)
	defer os.RemoveAll("chain1")
	if err != nil {
		t.Fatal(err)
	}

	c2, err := declChain("chain2", c1, 5, 8)
	defer os.RemoveAll("chain2")
	if err != nil {
		t.Fatal(err)
	}

	bestBlockHash := c2.BestBlockHash()
	if err := merge(c1, c2); err != nil {
		t.Fatal(err)
	}

	if *c1.BestBlockHash() != *bestBlockHash || *c2.BestBlockHash() != *bestBlockHash {
		t.Fatalf("test block sync failed")
	}

	if !c1.InMainChain(*bestBlockHash) || !c2.InMainChain(*bestBlockHash) {
		t.Fatalf("test block sync failed, best block is not in main chain")
	}
}

func TestDoubleSpentInDiffBlock(t *testing.T) {
	chainDB := dbm.NewDB("tx_pool_test", "leveldb", "tx_pool_test")
	defer os.RemoveAll("tx_pool_test")
	chain, _, txPool, err := MockChain(chainDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := AppendBlocks(chain, 6); err != nil {
		t.Fatal(err)
	}

	// create tx spend the coinbase output in block 1
	block, err := chain.GetBlockByHeight(1)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := CreateTxFromTx(block.Transactions[0], 0, 10000, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	newBlock, err := NewBlock(chain, []*types.Tx{tx}, []byte{byte(vm.OP_TRUE)})
	err = SolveAndUpdate(chain, newBlock)
	if err != nil {
		t.Fatal(err)
	}

	// create a double spent tx in another block
	tx, err = CreateTxFromTx(block.Transactions[0], 0, 10000, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	_, err = chain.ValidateTx(tx)
	if err == nil {
		t.Fatal("validate double spent tx success")
	}
	if txPool.HaveTransaction(&tx.ID) {
		t.Fatalf("tx pool have double spent tx")
	}
}

func TestDoubleSpentInSameBlock(t *testing.T) {
	chainDB := dbm.NewDB("tx_pool_test", "leveldb", "tx_pool_test")
	defer os.RemoveAll("tx_pool_test")
	chain, _, txPool, err := MockChain(chainDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := AppendBlocks(chain, 7); err != nil {
		t.Fatal(err)
	}

	// create tx spend the coinbase output in block 1
	block, err := chain.GetBlockByHeight(1)
	if err != nil {
		t.Fatal(err)
	}
	tx1, err := CreateTxFromTx(block.Transactions[0], 0, 10000, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	// create tx spend the coinbase output in block 1
	tx2, err := CreateTxFromTx(block.Transactions[0], 0, 10000, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	_, err = chain.ValidateTx(tx1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = chain.ValidateTx(tx2)
	if err != nil {
		t.Fatal(err)
	}

	if !txPool.HaveTransaction(&tx1.ID) {
		t.Fatalf("can't find tx in tx pool")
	}
	if !txPool.HaveTransaction(&tx2.ID) {
		t.Fatalf("can't find tx in tx pool")
	}

	block, err = NewBlock(chain, []*types.Tx{tx1, tx2}, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	if err := SolveAndUpdate(chain, block); err == nil {
		t.Fatalf("process double spent tx success")
	}
}

func TestTxPoolDependencyTx(t *testing.T) {
	chainDB := dbm.NewDB("tx_pool_test", "leveldb", "tx_pool_test")
	defer os.RemoveAll("tx_pool_test")
	chain, _, txPool, err := MockChain(chainDB)
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

	tx, err := CreateTxFromTx(block.Transactions[0], 0, 5000000000, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	outputAmount := uint64(5000000000)
	txs := []*types.Tx{nil}
	txs[0] = tx
	for i := 1; i < 10; i++ {
		outputAmount -= 50000000
		tx, err := CreateTxFromTx(txs[i-1], 0, outputAmount, []byte{byte(vm.OP_TRUE)})
		if err != nil {
			t.Fatal(err)
		}
		txs = append(txs, tx)
	}

	// validate tx and put it into tx pool
	for _, tx := range txs {
		if _, err := chain.ValidateTx(tx); err != nil {
			t.Fatal(err)
		}
		if !txPool.HaveTransaction(&tx.ID) {
			t.Fatal("can't find tx in txpool")
		}
	}

	block, err = NewBlock(chain, txs, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	if err := SolveAndUpdate(chain, block); err != nil {
		t.Fatal("process dependency tx failed")
	}
}
