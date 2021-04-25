package contract

import (
	"encoding/json"
	"sync"

	dbm "github.com/bytom/bytom/database/leveldb"
	chainjson "github.com/bytom/bytom/encoding/json"
	"github.com/bytom/bytom/errors"
)

var (
	userContractPrefix = []byte("UC:")
)

// pre-define errors for supporting bytom errorFormatter
var (
	ErrContractDuplicated = errors.New("contract is duplicated")
	ErrContractNotFound   = errors.New("contract not found")
)

// userContractKey return user contract key
func userContractKey(hash chainjson.HexBytes) []byte {
	return append(userContractPrefix, hash[:]...)
}

// Registry tracks and stores all user contract.
type Registry struct {
	db         dbm.DB
	contractMu sync.Mutex
}

//NewRegistry create new registry
func NewRegistry(db dbm.DB) *Registry {
	return &Registry{
		db: db,
	}
}

//Contract describe user contract
type Contract struct {
	Hash            chainjson.HexBytes `json:"id"`
	Alias           string             `json:"alias"`
	Contract        chainjson.HexBytes `json:"contract"`
	CallProgram     chainjson.HexBytes `json:"call_program"`
	RegisterProgram chainjson.HexBytes `json:"register_program"`
}

// SaveContract save user contract
func (reg *Registry) SaveContract(contract *Contract) error {
	reg.contractMu.Lock()
	defer reg.contractMu.Unlock()

	contractKey := userContractKey(contract.Hash)
	if existContract := reg.db.Get(contractKey); existContract != nil {
		return ErrContractDuplicated
	}

	rawContract, err := json.Marshal(contract)
	if err != nil {
		return err
	}

	reg.db.Set(contractKey, rawContract)
	return nil
}

//UpdateContract updates user contract alias
func (reg *Registry) UpdateContract(hash chainjson.HexBytes, alias string) error {
	reg.contractMu.Lock()
	defer reg.contractMu.Unlock()

	contract, err := reg.GetContract(hash)
	if err != nil {
		return err
	}

	contract.Alias = alias
	rawContract, err := json.Marshal(contract)
	if err != nil {
		return err
	}

	reg.db.Set(userContractKey(hash), rawContract)
	return nil
}

// GetContract get user contract
func (reg *Registry) GetContract(hash chainjson.HexBytes) (*Contract, error) {
	contract := &Contract{}
	if rawContract := reg.db.Get(userContractKey(hash)); rawContract != nil {
		return contract, json.Unmarshal(rawContract, contract)
	}
	return nil, ErrContractNotFound
}

// ListContracts returns user contracts
func (reg *Registry) ListContracts() ([]*Contract, error) {
	contracts := []*Contract{}
	contractIter := reg.db.IteratorPrefix(userContractPrefix)
	defer contractIter.Release()

	for contractIter.Next() {
		contract := &Contract{}
		if err := json.Unmarshal(contractIter.Value(), contract); err != nil {
			return nil, err
		}

		contracts = append(contracts, contract)
	}
	return contracts, nil
}
