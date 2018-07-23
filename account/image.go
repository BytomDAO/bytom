// Package account stores and tracks accounts within a Bytom Core.
package account

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/common"
)

// ImageSlice record info of single account
type ImageSlice struct {
	Account       *Account `json:"account"`
	ContractIndex uint64   `json:"contract_index"`
}

// Image is the struct for hold export account data
type Image struct {
	Slice []*ImageSlice `json:"slices"`
}

// Backup export all the account info into image
func (m *Manager) Backup() (*Image, error) {
	image := &Image{
		Slice: []*ImageSlice{},
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
	maxAccountIndex := uint64(0)
	storeBatch := m.db.NewBatch()
	for _, slice := range image.Slice {
		if existed := m.db.Get(Key(slice.Account.ID)); existed != nil {
			log.WithFields(log.Fields{
				"alias": slice.Account.Alias,
				"id":    slice.Account.ID,
			}).Warning("skip restore account due to already existed")
			continue
		}
		if existed := m.db.Get(aliasKey(slice.Account.Alias)); existed != nil {
			return ErrDuplicateAlias
		}

		rawAccount, err := json.Marshal(slice.Account)
		if err != nil {
			return ErrMarshalAccount
		}

		if slice.Account.Signer.KeyIndex > maxAccountIndex {
			maxAccountIndex = slice.Account.Signer.KeyIndex
		}
		storeBatch.Set(Key(slice.Account.ID), rawAccount)
		storeBatch.Set(aliasKey(slice.Account.Alias), []byte(slice.Account.ID))
	}

	if localIndex := m.getNextAccountIndex(); localIndex < maxAccountIndex {
		storeBatch.Set(accountIndexKey, common.Unit64ToBytes(maxAccountIndex))
	}
	storeBatch.Write()

	for _, slice := range image.Slice {
		for i := uint64(1); i <= slice.ContractIndex; i++ {
			if _, err := m.createAddress(slice.Account, false); err != nil {
				return err
			}
		}
	}
	return nil
}
