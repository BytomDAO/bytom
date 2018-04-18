// Package pseudohsm provides a pseudo HSM for development environments.
package pseudohsm

import (
	"io/ioutil"
	"path/filepath"
)

// KeyImage is the struct for hold export key data
type KeyImage struct {
	XPub XPub   `json:"xpub"`
	XKey []byte `json:"xkey"`
}

// Backup export all the HSM keys into array
func (h *HSM) Backup() ([]*KeyImage, error) {
	images := []*KeyImage{}
	xpubs := h.cache.keys()
	for _, xpub := range xpubs {
		xKey, err := ioutil.ReadFile(xpub.File)
		if err != nil {
			return nil, err
		}

		images = append(images, &KeyImage{XPub: xpub, XKey: xKey})
	}
	return images, nil
}

// Restore import the keyImages into HSM
func (h *HSM) Restore(images []*KeyImage) error {
	for _, image := range images {
		if ok := h.cache.hasAlias(image.XPub.Alias); ok {
			return ErrDuplicateKeyAlias
		}

		fileName := filepath.Base(image.XPub.File)
		image.XPub.File = h.keyStore.JoinPath(fileName)
		if err := writeKeyFile(image.XPub.File, image.XKey); err != nil {
			return nil
		}
		h.cache.add(image.XPub)
	}
	return nil
}
