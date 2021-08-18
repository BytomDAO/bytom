package test

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	"github.com/bytom/bytom/crypto/sha3pool"
	"github.com/bytom/bytom/database"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
	"github.com/bytom/bytom/protocol/validation"
	"github.com/bytom/bytom/protocol/vm"
)

func TestRegisterContract(t *testing.T) {
	db := dbm.NewDB("contract_test_db", "leveldb", "contract_test_db")
	defer os.RemoveAll("contract_test_db")

	chain, _, _, _ := MockChain(db)
	block, err := NewBlock(chain, nil, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	contract, err := hex.DecodeString("0164740a52797b937b788791698700c0")
	if err != nil {
		t.Fatal(err)
	}

	tx, err := CreateRegisterContractTx(block.Transactions[0], 0, contract)
	if err != nil {
		t.Fatal(err)
	}

	defaultCtrlProg := []byte{byte(vm.OP_TRUE)}
	block, err = NewBlock(chain, []*types.Tx{tx}, defaultCtrlProg)
	if err != nil {
		t.Fatal(err)
	}

	contractView := state.NewContractViewpoint()
	if err := contractView.ApplyBlock(block); err != nil {
		t.Fatal(err)
	}

	utxoView := &state.UtxoViewpoint{}
	store := database.NewStore(db)
	if err := store.SaveChainStatus(&block.BlockHeader, []*types.BlockHeader{&block.BlockHeader}, utxoView, contractView, 0, &bc.Hash{}); err != nil {
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

func TestValidateContract(t *testing.T) {
	// register
	db := dbm.NewDB("contract_test_db", "leveldb", "contract_test_db")
	defer os.RemoveAll("contract_test_db")

	store := database.NewStore(db)
	chain, _, _, _ := MockChainWithStore(store)
	block, err := NewBlock(chain, nil, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	contract, err := hex.DecodeString("0164740a52797b937b788791698700c0")
	if err != nil {
		t.Fatal(err)
	}

	tx, err := CreateRegisterContractTx(block.Transactions[0], 0, contract)
	if err != nil {
		t.Fatal(err)
	}

	defaultCtrlProg := []byte{byte(vm.OP_TRUE)}
	block, err = NewBlock(chain, []*types.Tx{tx}, defaultCtrlProg)
	if err != nil {
		t.Fatal(err)
	}

	contractView := state.NewContractViewpoint()
	if err := contractView.ApplyBlock(block); err != nil {
		t.Fatal(err)
	}

	utxoView := &state.UtxoViewpoint{}
	if err := store.SaveChainStatus(&block.BlockHeader, []*types.BlockHeader{&block.BlockHeader}, utxoView, contractView, 0, &bc.Hash{}); err != nil {
		t.Fatal(err)
	}

	// call
	block, err = NewBlock(chain, nil, []byte{byte(vm.OP_TRUE)})
	if err != nil {
		t.Fatal(err)
	}

	var hash [32]byte
	sha3pool.Sum256(hash[:], contract)
	tx, err = CreateCallContractTx(block.Transactions[0], 0, hash[:])
	if err != nil {
		t.Fatal(err)
	}

	prevBlock, err := NewBlock(chain, []*types.Tx{tx}, defaultCtrlProg)
	if err != nil {
		t.Fatal(err)
	}

	// use valid arguments
	arguments := [][]byte{
		{byte(99)},
		{byte(1)},
	}
	tx, err = CreateUseContractTx(prevBlock.Transactions[1], 0, arguments, defaultCtrlProg)
	if err != nil {
		t.Fatal(err)
	}

	block, err = NewBlock(chain, []*types.Tx{tx}, defaultCtrlProg)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := validation.ValidateTx(tx.Tx, types.MapBlock(block), chain.ProgramConverter); err != nil {
		t.Fatal(err)
	}

	// use invalid arguments
	arguments = [][]byte{
		{byte(99)},
		{byte(2)},
	}
	tx, err = CreateUseContractTx(prevBlock.Transactions[1], 0, arguments, defaultCtrlProg)
	if err != nil {
		t.Fatal(err)
	}

	block, err = NewBlock(chain, []*types.Tx{tx}, defaultCtrlProg)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := validation.ValidateTx(tx.Tx, types.MapBlock(block), chain.ProgramConverter); errors.Root(err) != vm.ErrFalseVMResult {
		t.Fatal(err)
	}
}
