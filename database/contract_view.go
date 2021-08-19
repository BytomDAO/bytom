package database

import (
	"bytes"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/state"
)

// CalcContractKey calculate contract key
func CalcContractKey(hash [32]byte) []byte {
	return append(ContractPrefix, hash[:]...)
}

func saveContractView(db dbm.DB, batch dbm.Batch, view *state.ContractViewpoint) error {
	for hash, value := range view.AttachEntries {
		data := db.Get(CalcContractKey(hash))
		// contract is not exist
		if data == nil {
			// key:"c:sha256(program.Code)" value:"txID+program.Code"
			batch.Set(CalcContractKey(hash), value)
		}
		// contract is deleted in the same batch
		if v, ok := view.DetachEntries[hash]; ok && bytes.Equal(data, v) {
			batch.Set(CalcContractKey(hash), value)
		}
	}
	return nil
}

func deleteContractView(db dbm.DB, batch dbm.Batch, view *state.ContractViewpoint) error {
	for hash, value := range view.DetachEntries {
		// rollback is forbidden if contract register transaction id is different
		if bytes.Equal(db.Get(CalcContractKey(hash)), value) {
			batch.Delete(CalcContractKey(hash))
		}
	}
	return nil
}

func getContract(db dbm.DB, hash [32]byte) ([]byte, error) {
	data := db.Get(CalcContractKey(hash))
	if data == nil {
		return nil, errors.New("can't find the registered contract by contract hash")
	}

	// data:"txID+program.Code" len(txID) == 32
	if len(data) <= 32 {
		return nil, errors.New("registered contract format error")
	}

	return data[32:], nil
}
