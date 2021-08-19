package chainmgr

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/protocol/bc/types"
)

func TestReadWriteBlocks(t *testing.T) {
	tmp, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	testDB := dbm.NewDB("testdb", "leveldb", tmp)
	defer testDB.Close()

	s := newStorage(testDB)

	cases := []struct {
		storageRAMLimit int
		blocks          []*types.Block
		peerID          string
		isRAM           bool
	}{
		{
			storageRAMLimit: 800 * 1024 * 1024,
			blocks:          mockBlocks(nil, 500),
			peerID:          "testPeer",
			isRAM:           true,
		},
		{
			storageRAMLimit: 1,
			blocks:          mockBlocks(nil, 500),
			peerID:          "testPeer",
			isRAM:           false,
		},
	}

	for index, c := range cases {
		maxByteOfStorageRAM = c.storageRAMLimit
		s.writeBlocks(c.peerID, c.blocks)

		for i := 0; i < len(c.blocks); i++ {
			blockStorage, err := s.readBlock(uint64(i))
			if err != nil {
				t.Fatal(err)
			}

			if blockStorage.isRAM != c.isRAM {
				t.Fatalf("case %d: TestReadWriteBlocks block %d isRAM: got %t want %t", index, i, blockStorage.isRAM, c.isRAM)
			}

			if blockStorage.block.Hash() != c.blocks[i].Hash() {
				t.Fatalf("case %d: TestReadWriteBlocks block %d: got %s want %s", index, i, spew.Sdump(blockStorage.block), spew.Sdump(c.blocks[i]))
			}
		}
	}
}

func TestDeleteBlock(t *testing.T) {
	tmp, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	testDB := dbm.NewDB("testdb", "leveldb", tmp)
	defer testDB.Close()

	maxByteOfStorageRAM = 1024
	blocks := mockBlocks(nil, 500)
	s := newStorage(testDB)
	for i, block := range blocks {
		if err := s.writeBlocks("testPeer", []*types.Block{block}); err != nil {
			t.Fatal(err)
		}

		blockStorage, err := s.readBlock(block.Height)
		if err != nil {
			t.Fatal(err)
		}

		if !blockStorage.isRAM {
			t.Fatalf("TestReadWriteBlocks block %d isRAM: got %t want %t", i, blockStorage.isRAM, true)
		}

		s.deleteBlock(block.Height)
	}

}

func TestLevelDBStorageReadWrite(t *testing.T) {
	tmp, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	testDB := dbm.NewDB("testdb", "leveldb", tmp)
	defer testDB.Close()

	blocks := mockBlocks(nil, 16)
	s := newDBStore(testDB)

	for i, block := range blocks {
		err := s.writeBlock(block)
		if err != nil {
			t.Fatal(err)
		}

		gotBlock, err := s.readBlock(block.Height)
		if err != nil {
			t.Fatal(err)
		}

		if gotBlock.Hash() != block.Hash() {
			t.Fatalf("TestLevelDBStorageReadWrite block %d: got %s want %s", i, spew.Sdump(gotBlock), spew.Sdump(block))
		}

		s.clearData()
		_, err = s.readBlock(block.Height)
		if err == nil {
			t.Fatalf("TestLevelDBStorageReadWrite clear data err block %d", i)
		}
	}
}
