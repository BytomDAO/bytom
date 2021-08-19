package test

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/bytom/bytom/crypto/sha3pool"
	"github.com/bytom/bytom/database"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/vm"
)

func TestRegisterContract(t *testing.T) {
	dbName := uuid.New().String()
	db := dbm.NewDB(dbName, "leveldb", dbName)
	defer os.RemoveAll(dbName)

	store := database.NewStore(db)
	chain, _, _, _ := mockChainWithStore(store)

	contract, err := hex.DecodeString("0164740a52797b937b788791698700c0")
	if err != nil {
		t.Fatal(err)
	}

	if err := registerContract(chain, store, contract); err != nil {
		t.Fatal(err)
	}

	var hash [32]byte
	sha3pool.Sum256(hash[:], contract)
	data := db.Get(database.CalcContractKey(hash))
	if data == nil {
		t.Fatalf("can't find the registered contract by contract hash %v", hash)
	}

	if !bytes.Equal(contract, data[32:]) {
		t.Fatalf("TestRegisterContract failed: got %v want %v", data[32:], contract)
	}
}

func TestUseContractSuccess(t *testing.T) {
	dbName := uuid.New().String()
	db := dbm.NewDB(dbName, "leveldb", dbName)
	defer os.RemoveAll(dbName)

	store := database.NewStore(db)
	chain, _, _, _ := mockChainWithStore(store)

	contract, err := hex.DecodeString("0164740a52797b937b788791698700c0")
	if err != nil {
		t.Fatal(err)
	}

	if err := registerContract(chain, store, contract); err != nil {
		t.Fatal(err)
	}

	arguments := [][]byte{
		{byte(99)},
		{byte(1)},
	}

	if err := validateContract(chain, contract, arguments, nil); err != nil {
		t.Fatal(err)
	}
}

func TestUseContractFailed(t *testing.T) {
	dbName := uuid.New().String()
	db := dbm.NewDB(dbName, "leveldb", dbName)
	defer os.RemoveAll(dbName)

	store := database.NewStore(db)
	chain, _, _, _ := mockChainWithStore(store)

	contract, err := hex.DecodeString("0164740a52797b937b788791698700c0")
	if err != nil {
		t.Fatal(err)
	}

	if err := registerContract(chain, store, contract); err != nil {
		t.Fatal(err)
	}

	arguments := [][]byte{
		{byte(99)},
		{byte(2)},
	}

	if err := validateContract(chain, contract, arguments, nil); errors.Root(err) != vm.ErrFalseVMResult {
		t.Fatal(err)
	}
}

func TestUseContractWithStateDataSuccess(t *testing.T) {
	dbName := uuid.New().String()
	db := dbm.NewDB(dbName, "leveldb", dbName)
	defer os.RemoveAll(dbName)

	store := database.NewStore(db)
	chain, _, _, _ := mockChainWithStore(store)

	contract, err := hex.DecodeString("01646c7c740a52797b937b788791698700c0")
	if err != nil {
		t.Fatal(err)
	}

	if err := registerContract(chain, store, contract); err != nil {
		t.Fatal(err)
	}

	arguments := [][]byte{
		{byte(1)},
	}

	stateData := [][]byte{
		{byte(99)},
	}

	if err := validateContract(chain, contract, arguments, stateData); err != nil {
		t.Fatal(err)
	}
}

func TestUseContractWithStateDataFailed(t *testing.T) {
	dbName := uuid.New().String()
	db := dbm.NewDB(dbName, "leveldb", dbName)
	defer os.RemoveAll(dbName)

	store := database.NewStore(db)
	chain, _, _, _ := mockChainWithStore(store)

	contract, err := hex.DecodeString("01646c7c740a52797b937b788791698700c0")
	if err != nil {
		t.Fatal(err)
	}

	if err := registerContract(chain, store, contract); err != nil {
		t.Fatal(err)
	}

	arguments := [][]byte{
		{byte(2)},
	}

	stateData := [][]byte{
		{byte(99)},
	}

	if err := validateContract(chain, contract, arguments, stateData); errors.Root(err) != vm.ErrFalseVMResult {
		t.Fatal(err)
	}
}
