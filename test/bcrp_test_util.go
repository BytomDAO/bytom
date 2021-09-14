package test

import (
	"os"

	"github.com/bytom/bytom/contract"
	"github.com/bytom/bytom/crypto/sha3pool"
	"github.com/bytom/bytom/database"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
	"github.com/bytom/bytom/protocol/validation"
	"github.com/bytom/bytom/protocol/vm"
)

func mockChainWithStore(store *database.Store) (*protocol.Chain, *database.Store, *protocol.TxPool, error) {
	traceDB := dbm.NewDB("trace", "leveldb", "trace_db")
	defer os.RemoveAll("trace_db")

	dispatcher := event.NewDispatcher()
	traceStore := contract.NewTraceStore(traceDB)
	txPool := protocol.NewTxPool(store, dispatcher)
	chain, err := protocol.NewChain(store, traceStore, txPool, dispatcher)
	return chain, store, txPool, err
}

func registerContract(chain *protocol.Chain, store *database.Store, contract []byte) error {
	tx, err := CreateRegisterContractTx(contract)
	if err != nil {
		return err
	}

	defaultCtrlProg := []byte{byte(vm.OP_TRUE)}
	block, err := NewBlock(chain, []*types.Tx{tx}, defaultCtrlProg)
	if err != nil {
		return err
	}

	contractView := state.NewContractViewpoint()
	if err := contractView.ApplyBlock(block); err != nil {
		return err
	}

	utxoView := &state.UtxoViewpoint{}
	return store.SaveChainStatus(&block.BlockHeader, []*types.BlockHeader{&block.BlockHeader}, utxoView, contractView, 0, &bc.Hash{})
}

func validateContract(chain *protocol.Chain, contract []byte, arguments [][]byte, stateData [][]byte) error {
	var hash [32]byte
	sha3pool.Sum256(hash[:], contract)

	tx, err := CreateUseContractTx(hash[:], arguments, stateData)
	if err != nil {
		return err
	}

	defaultCtrlProg := []byte{byte(vm.OP_TRUE)}
	block, err := NewBlock(chain, []*types.Tx{tx}, defaultCtrlProg)
	if err != nil {
		return err
	}

	_, err = validation.ValidateTx(tx.Tx, types.MapBlock(block), chain.ProgramConverter)
	return err
}
