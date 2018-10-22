// Package account stores and tracks accounts within a Bytom Core.
package account

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/signers"
	"github.com/bytom/common"
)

// ImageSlice record info of single account
type ImageSlice struct {
	Account       *Account `json:"account"`
	ContractIndex uint64   `json:"contract_index"`
	Bip44IntIndex uint64   `json:"contract_internal"`
	Bip44ExtIndex uint64   `json:"contract_external"`
}

// Image is the struct for hold export account data
type Image struct {
	Slice []*ImageSlice `json:"slices"`
}

// Backup export all the account info into image
func (m *Manager) Backup() (*Image, error) {
	m.accountMu.Lock()
	defer m.accountMu.Unlock()
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
			ContractIndex: m.GetContractIndex(a.ID),
			Bip44IntIndex: m.GetBip44ContractIndex(a.ID, true),
			Bip44ExtIndex: m.GetBip44ContractIndex(a.ID, false),
		})
	}
	return image, nil
}

// Restore import the accountImages into account manage
func (m *Manager) Restore(image *Image) error {
	m.accountMu.Lock()
	defer m.accountMu.Unlock()
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

		storeBatch.Set(Key(slice.Account.ID), rawAccount)
		storeBatch.Set(aliasKey(slice.Account.Alias), []byte(slice.Account.ID))
		index := uint64(0)
		if rawIndexBytes := m.db.Get(GetAccountIndexKey(slice.Account.XPubs)); rawIndexBytes != nil {
			index = common.BytesToUnit64(rawIndexBytes)
		}
		if index < slice.Account.KeyIndex && slice.Account.DeriveRule == signers.BIP0044 {
			storeBatch.Set(GetAccountIndexKey(slice.Account.XPubs), common.Unit64ToBytes(slice.Account.KeyIndex))
		}
	}

	storeBatch.Write()

	for _, slice := range image.Slice {
		for i := uint64(1); i <= slice.ContractIndex; i++ {
			if _, err := m.createAddress(slice.Account, false); err != nil {
				return err
			}
		}
		for i := uint64(1); i <= slice.Bip44IntIndex; i++ {
			if _, err := m.createAddress(slice.Account, true); err != nil {
				return err
			}
		}
		for i := uint64(1); i <= slice.Bip44ExtIndex; i++ {
			if _, err := m.createAddress(slice.Account, false); err != nil {
				return err
			}
		}
	}
	return nil
}
