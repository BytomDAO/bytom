// Package account stores and tracks accounts within a Chain Core.
package account

import (
	"encoding/json"

	"github.com/bytom/common"
)

// ImageSlice record info of single account
type ImageSlice struct {
	Account       *Account `json:"account"`
	ContractIndex uint64   `json:"contract_index"`
}

// Image is the struct for hold export account data
type Image struct {
	Slice        []*ImageSlice `json:"slices"`
	AccountIndex uint64        `json:"account_index"`
}

// Backup export all the account info into image
func (m *Manager) Backup() (*Image, error) {
	image := &Image{
		Slice:        []*ImageSlice{},
		AccountIndex: m.getNextAccountIndex(),
	}

	accountIter := m.db.IteratorPrefix(accountPrefix)
	defer accountIter.Release()
	for accountIter.Next() {
		a := &Account{}
		if err := json.Unmarshal(accountIter.Value(), a); err != nil {
			return nil, err
		}

		image.Slice = append(image.Slice, &ImageSlice{
			Account:       a,
			ContractIndex: m.getNextContractIndex(a.ID),
		})
	}
	return image, nil
}

// Restore import the accountImages into account manage
func (m *Manager) Restore(image *Image) error {
	storeBatch := m.db.NewBatch()
	for _, slice := range image.Slice {
		if existed := m.db.Get(aliasKey(slice.Account.Alias)); existed != nil {
			return ErrDuplicateAlias
		}

		rawAccount, err := json.Marshal(slice.Account)
		if err != nil {
			return ErrMarshalAccount
		}

		storeBatch.Set(Key(slice.Account.ID), rawAccount)
		storeBatch.Set(aliasKey(slice.Account.Alias), []byte(slice.Account.ID))
		storeBatch.Set(contractIndexKey(slice.Account.ID), common.Unit64ToBytes(slice.ContractIndex))
	}

	if localIndex := m.getNextAccountIndex(); localIndex < image.AccountIndex {
		storeBatch.Set(accountIndexKey, common.Unit64ToBytes(image.AccountIndex))
	}
	storeBatch.Write()

	for _, slice := range image.Slice {
		for i := uint64(1); i < slice.ContractIndex; i++ {
			if _, err := m.createAddress(nil, slice.Account, false); err != nil {
				return err
			}
		}
	}
	return nil
}
