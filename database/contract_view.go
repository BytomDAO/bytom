package database

import (
	"bytes"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/protocol/state"
)

// ContractPrefix contract prefix
var ContractPrefix = []byte("C:")

// CalcContractKey calculate contract key
func CalcContractKey(hash [32]byte) []byte {
	return append(ContractPrefix, hash[:]...)
}

func saveContractView(db dbm.DB, batch dbm.Batch, view *state.ContractViewpoint) error {
	for hash, value := range view.AttachEntries {
		// contract exist, overwriting is forbidden
		if db.Get(CalcContractKey(hash)) == nil {
			// key:"c:sha256(program.Code)" value:"txID+program.Code"
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
