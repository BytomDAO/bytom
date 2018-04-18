package asset

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/common"
)

// Image is the struct for hold export asset data
type Image struct {
	Assets     []*Asset `json:"assets"`
	AssetIndex uint64   `json:"asset_index"`
}

// Backup export all the asset info into image
func (reg *Registry) Backup() (*Image, error) {
	assetImage := &Image{
		AssetIndex: reg.getNextAssetIndex(),
		Assets:     []*Asset{},
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
	storeBatch := reg.db.NewBatch()
	for _, asset := range image.Assets {
		if localAssetID := reg.db.Get(AliasKey(*asset.Alias)); localAssetID != nil {
			if string(localAssetID) != asset.AssetID.String() {
				return ErrDuplicateAlias
			}

			log.WithFields(log.Fields{"alias": asset.Alias, "id": asset.AssetID}).Warning("skip restore asset due to already existed")
			continue
		}

		rawAsset, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		storeBatch.Set(AliasKey(*asset.Alias), asset.AssetID.Bytes())
		storeBatch.Set(Key(&asset.AssetID), rawAsset)
	}

	if localIndex := reg.getNextAssetIndex(); localIndex < image.AssetIndex {
		storeBatch.Set(assetIndexKey, common.Unit64ToBytes(image.AssetIndex))
	}
	storeBatch.Write()
	return nil
}
