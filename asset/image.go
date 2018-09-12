package asset

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/common"
)

// Image is the struct for hold export asset data
type Image struct {
	Assets []*Asset `json:"assets"`
}

// Backup export all the asset info into image
func (reg *Registry) Backup() (*Image, error) {
	assetImage := &Image{
		Assets: []*Asset{},
	}

	assetIter := reg.db.IteratorPrefix([]byte(assetPrefix))
	defer assetIter.Release()
	for assetIter.Next() {
		asset := &Asset{}
		if err := json.Unmarshal(assetIter.Value(), asset); err != nil {
			return nil, err
		}
		assetImage.Assets = append(assetImage.Assets, asset)
	}

	return assetImage, nil
}

// Restore load the image data into asset manage
func (reg *Registry) Restore(image *Image) error {
	maxAssetIndex := uint64(0)
	storeBatch := reg.db.NewBatch()
	for _, asset := range image.Assets {
		if existed := reg.db.Get(Key(&asset.AssetID)); existed != nil {
			log.WithFields(log.Fields{"alias": asset.Alias, "id": asset.AssetID}).Warning("skip restore asset due to already existed")
			continue
		}
		if existed := reg.db.Get(aliasKey(*asset.Alias)); existed != nil {
			return ErrDuplicateAlias
		}

		rawAsset, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		if asset.Signer.KeyIndex > maxAssetIndex {
			maxAssetIndex = asset.Signer.KeyIndex
		}
		storeBatch.Set(aliasKey(*asset.Alias), []byte(asset.AssetID.String()))
		storeBatch.Set(Key(&asset.AssetID), rawAsset)
	}

	if localIndex := reg.getNextAssetIndex(); localIndex < maxAssetIndex {
		storeBatch.Set(assetIndexKey, common.Unit64ToBytes(maxAssetIndex))
	}
	storeBatch.Write()
	return nil
}
