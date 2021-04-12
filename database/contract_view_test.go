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
	testDB  dbm.DB
	program []byte
	hash    [32]byte
	txID1   *bc.Hash
	txID2   *bc.Hash
)

func init() {
	testDB = dbm.NewDB("testdb", "leveldb", "temp")
	contract := "6a4c04626372704c01014c2820e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c0"
	program, _ = hex.DecodeString(contract)
	sha3pool.Sum256(hash[:], program)
	txID1 = &bc.Hash{V0: 1, V1: 1, V2: 1, V3: 1}
	txID2 = &bc.Hash{V0: 2, V1: 2, V2: 2, V3: 2}
	registerByTx(txID1)
}

func registerByTx(txID *bc.Hash) {
	batch := testDB.NewBatch()
	contractView := state.NewContractViewpoint()
	contractView.AttachEntries[hash] = append(txID.Bytes(), program...)
	if err := saveContractView(testDB, batch, contractView); err != nil {
		panic(err)
	}

	batch.Write()
}

func rollbackByTx(txID *bc.Hash) {
	batch := testDB.NewBatch()
	contractView := state.NewContractViewpoint()
	contractView.DetachEntries[hash] = append(txID.Bytes(), program...)
	if err := deleteContractView(testDB, batch, contractView); err != nil {
		panic(err)
	}

	batch.Write()
}

func TestRollbackAndRegisterAgain(t *testing.T) {
	defer os.RemoveAll("temp")

	// only rollback
	rollbackByTx(txID1)
	data := testDB.Get(CalcContractKey(hash))
	if data != nil {
		t.Errorf("registered contract should be deleted")
	}

	// register again
	registerByTx(txID1)
	data = testDB.Get(CalcContractKey(hash))
	if data == nil {
		t.Errorf("can't find the registered contract by contract hash %v", hash)
	}

	expect := append(txID1.Bytes(), program...)
	if !bytes.Equal(data, expect) {
		t.Errorf("got contract data: %v, expect contract data: %v", data, expect)
	}
}

func TestRollbackAndRegisterByAnotherTx(t *testing.T) {
	defer os.RemoveAll("temp")

	rollbackByTx(txID1)
	data := testDB.Get(CalcContractKey(hash))
	if data != nil {
		t.Errorf("registered contract should be deleted")
	}

	// register by another transaction
	registerByTx(txID2)
	data = testDB.Get(CalcContractKey(hash))
	if data == nil {
		t.Errorf("can't find the registered contract by contract hash %v", hash)
	}

	expect := append(txID2.Bytes(), program...)
	if !bytes.Equal(data, expect) {
		t.Errorf("got contract data: %v, expect contract data: %v", data, expect)
	}
}

func TestRepeatRegisterAndRollback(t *testing.T) {
	defer os.RemoveAll("temp")

	// repeat register
	registerByTx(txID2)
	data := testDB.Get(CalcContractKey(hash))
	if data == nil {
		t.Errorf("can't find the registered contract by contract hash %v", hash)
	}

	expect := append(txID1.Bytes(), program...)
	if !bytes.Equal(data, expect) {
		t.Errorf("got contract data: %v, expect contract data: %v", data, expect)
	}

	// rollback by repeat register transaction
	rollbackByTx(txID2)
	data = testDB.Get(CalcContractKey(hash))
	if data == nil {
		t.Errorf("can't find the registered contract by contract hash %v", hash)
	}

	expect = append(txID1.Bytes(), program...)
	if !bytes.Equal(data, expect) {
		t.Errorf("got contract data: %v, expect contract data: %v", data, expect)
	}
}
