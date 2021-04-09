package database

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	"github.com/bytom/bytom/crypto/sha3pool"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/state"
)

var (
	testDB       dbm.DB
	contractView *state.ContractViewpoint
	hash         [32]byte
	program      []byte
)

func init() {
	testDB = dbm.NewDB("testdb", "leveldb", "temp")
	contractView = state.NewContractViewpoint()
	contract := "6a4c04626372704c01014c2820e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c0"
	program, _ = hex.DecodeString(contract)
	sha3pool.Sum256(hash[:], program)
}

func TestSaveContractView(t *testing.T) {
	defer os.RemoveAll("temp")

	txID := &bc.Hash{V0: 0, V1: 1, V2: 2, V3: 3}
	contractView.AttachEntries[hash] = append(txID.Bytes(), program...)

	batch := testDB.NewBatch()
	if err := saveContractView(testDB, batch, contractView); err != nil {
		t.Fatal(err)
	}
	batch.Write()

	txID1 := &bc.Hash{V0: 1, V1: 1, V2: 1, V3: 1}
	contractView.AttachEntries[hash] = append(txID1.Bytes(), program...)

	batch = testDB.NewBatch()
	if err := saveContractView(testDB, batch, contractView); err != nil {
		t.Fatal(err)
	}
	batch.Write()

	data := testDB.Get(CalcContractKey(hash))
	if data == nil {
		t.Errorf("can't find the registered contract by contract hash %v", hash)
	}

	expect := append(txID.Bytes(), program...)
	if !bytes.Equal(data, expect) {
		t.Errorf("got contract data: %v, expect contract data: %v", data, expect)
	}
}

func TestDeleteContractView(t *testing.T) {
	defer os.RemoveAll("temp")

	txID := &bc.Hash{V0: 0, V1: 1, V2: 2, V3: 3}
	contractView.AttachEntries[hash] = append(txID.Bytes(), program...)

	batch := testDB.NewBatch()
	if err := saveContractView(testDB, batch, contractView); err != nil {
		t.Fatal(err)
	}
	batch.Write()

	txID1 := &bc.Hash{V0: 1, V1: 1, V2: 1, V3: 1}
	contractView.DetachEntries[hash] = append(txID1.Bytes(), program...)

	batch = testDB.NewBatch()
	if err := deleteContractView(testDB, batch, contractView); err != nil {
		t.Fatal(err)
	}
	batch.Write()

	data := testDB.Get(CalcContractKey(hash))
	if data == nil {
		t.Errorf("can't find the registered contract by contract hash %v", hash)
	}

	contractView.DetachEntries[hash] = append(txID.Bytes(), program...)
	batch = testDB.NewBatch()
	if err := deleteContractView(testDB, batch, contractView); err != nil {
		t.Fatal(err)
	}
	batch.Write()

	data = testDB.Get(CalcContractKey(hash))
	if data != nil {
		t.Errorf("registered contract should be deleted")
	}
}
