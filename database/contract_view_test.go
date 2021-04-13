package database

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/bytom/bytom/crypto/sha3pool"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/state"
)

var (
	program, _ = hex.DecodeString("6a4c04626372704c01014c2820e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c0")
	hash       [32]byte
	txID1      = &bc.Hash{V0: 1, V1: 1, V2: 1, V3: 1}
	txID2      = &bc.Hash{V0: 2, V1: 2, V2: 2, V3: 2}
)

func init() {
	sha3pool.Sum256(hash[:], program)
}

// register contract by transaction 1
func registerContract(testDB dbm.DB) {
	contractView := state.NewContractViewpoint()
	contractView.AttachEntries[hash] = append(txID1.Bytes(), program...)
	if err := setContractView(testDB, contractView); err != nil {
		panic(err)
	}
}

func setContractView(testDB dbm.DB, contractView *state.ContractViewpoint) error {
	batch := testDB.NewBatch()
	if err := deleteContractView(testDB, batch, contractView); err != nil {
		return err
	}

	if err := saveContractView(testDB, batch, contractView); err != nil {
		return err
	}

	batch.Write()
	return nil
}

func assertDBContractData(testDB dbm.DB, txID *bc.Hash, t *testing.T) {
	data := testDB.Get(CalcContractKey(hash))
	if data == nil {
		t.Errorf("can't find the registered contract by contract hash %v", hash)
	}

	expect := append(txID.Bytes(), program...)
	if !bytes.Equal(data, expect) {
		t.Errorf("got contract data: %v, expect contract data: %v", data, expect)
	}
}

func TestRollback(t *testing.T) {
	dbName := uuid.New().String()
	testDB := dbm.NewDB(dbName, "leveldb", dbName)
	registerContract(testDB)
	defer os.RemoveAll(dbName)

	contractView := state.NewContractViewpoint()
	// rollback
	contractView.DetachEntries[hash] = append(txID1.Bytes(), program...)
	if err := setContractView(testDB, contractView); err != nil {
		t.Errorf("set contract view failed")
	}

	data := testDB.Get(CalcContractKey(hash))
	if data != nil {
		t.Errorf("registered contract should be deleted")
	}
}

func TestRollbackAndRegisterAgain(t *testing.T) {
	dbName := uuid.New().String()
	testDB := dbm.NewDB(dbName, "leveldb", dbName)
	registerContract(testDB)
	defer os.RemoveAll(dbName)

	contractView := state.NewContractViewpoint()
	// rollback
	contractView.DetachEntries[hash] = append(txID1.Bytes(), program...)
	// register again
	contractView.AttachEntries[hash] = append(txID1.Bytes(), program...)
	if err := setContractView(testDB, contractView); err != nil {
		t.Errorf("set contract view failed")
	}

	assertDBContractData(testDB, txID1, t)
}

func TestRollbackAndRegisterByAnotherTx(t *testing.T) {
	dbName := uuid.New().String()
	testDB := dbm.NewDB(dbName, "leveldb", dbName)
	registerContract(testDB)
	defer os.RemoveAll(dbName)

	contractView := state.NewContractViewpoint()
	// rollback
	contractView.DetachEntries[hash] = append(txID1.Bytes(), program...)
	// register by another transaction
	contractView.AttachEntries[hash] = append(txID2.Bytes(), program...)
	if err := setContractView(testDB, contractView); err != nil {
		t.Errorf("set contract view failed")
	}

	assertDBContractData(testDB, txID2, t)
}

func TestRepeatRegisterAndRollback(t *testing.T) {
	dbName := uuid.New().String()
	testDB := dbm.NewDB(dbName, "leveldb", dbName)
	registerContract(testDB)
	defer os.RemoveAll(dbName)

	// repeat register
	contractView := state.NewContractViewpoint()
	contractView.AttachEntries[hash] = append(txID2.Bytes(), program...)
	if err := setContractView(testDB, contractView); err != nil {
		t.Errorf("set contract view failed")
	}

	assertDBContractData(testDB, txID1, t)

	// rollback by repeat register transaction
	contractView = state.NewContractViewpoint()
	contractView.DetachEntries[hash] = append(txID2.Bytes(), program...)
	if err := setContractView(testDB, contractView); err != nil {
		t.Errorf("set contract view failed")
	}

	assertDBContractData(testDB, txID1, t)
}
