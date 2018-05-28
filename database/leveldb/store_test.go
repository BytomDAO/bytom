package leveldb

import (
	"os"
	"testing"

	"github.com/bytom/config"
	"github.com/bytom/protocol/bc"

	dbm "github.com/tendermint/tmlibs/db"
)

func TestLoadBlockIndex(t *testing.T) {
	defer os.RemoveAll("temp")
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	store := NewStore(testDB)

	block := config.GenesisBlock()
	txStatus := bc.NewTransactionStatus()

	if err := store.SaveBlock(block, txStatus); err != nil {
		t.Fatal(err)
	}

	for block.Height <= 128 {
		preHash := block.Hash()
		block.PreviousBlockHash = preHash
		block.Height += 1
		if err := store.SaveBlock(block, txStatus); err != nil {
			t.Fatal(err)
		}

		if block.Height%32 != 0 {
			continue
		}

		for i := uint64(0); i < block.Height/32; i++ {
			block.Nonce += 1
			if err := store.SaveBlock(block, txStatus); err != nil {
				t.Fatal(err)
			}
		}
	}

	if _, err := store.LoadBlockIndex(); err != nil {
		t.Fatal(err)
	}
}
