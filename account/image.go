// Package account stores and tracks accounts within a Chain Core.
package account

import (
	"encoding/json"
)

type AccountSlice struct {
	Account       *Account
	ContractIndex uint64
}

type AccountImage struct {
	AccountSlice []*AccountSlice
	AccountIndex uint64
}

func (m *Manager) Backup() (*AccountImage, error) {
	accountSlices := []*AccountSlice{}
	accountIter := m.db.IteratorPrefix([]byte(accountPrefix))
	defer accountIter.Release()

	for accountIter.Next() {
		accountSlice := &AccountSlice{}
		if err := json.Unmarshal(accountIter.Value(), accountSlice.Account); err != nil {
			return nil, err
		}

		accountSlice.ContractIndex = m.getNextContractIndex(accountSlice.Account.ID)
		accountSlices = append(accountSlices, accountSlice)
	}

	accountImage := &AccountImage{
		AccountSlice: accountSlices,
		AccountIndex: m.getNextAccountIndex(),
	}
	return accountImage, nil
}

func (m *Manager) Restore(image *AccountImage) error {
	if localIndex := m.getNextAccountIndex(); localIndex > image.AccountIndex {
		image.AccountIndex = localIndex
	}

	storeBatch := m.db.NewBatch()
	for _, accountSlice := range image.AccountSlice {
		rawAccount, err := json.Marshal(accountSlice.Account)
		if err != nil {
			return ErrMarshalAccount
		}

		if existed := m.db.Get(aliasKey(accountSlice.Account.Alias)); existed != nil {
			return ErrDuplicateAlias
		}

		accountID := Key(accountSlice.Account.ID)
		storeBatch.Set(accountID, rawAccount)
		storeBatch.Set(aliasKey(accountSlice.Account.Alias), accountID)
		storeBatch.Set(contractIndexKey(accountSlice.Account.ID), convertUnit64ToBytes(accountSlice.ContractIndex))
	}

	storeBatch.Set(accountIndexKey, convertUnit64ToBytes(image.AccountIndex))
	storeBatch.Write()
	return nil
}
