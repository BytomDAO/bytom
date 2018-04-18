package asset

import (
	"encoding/json"
)

type AssetImage struct {
	Assets     []*Asset
	AssetIndex uint64
}

func (reg *Registry) Backup() (*AssetImage, error) {
	assetIndex, err := reg.getNextAssetIndex()
	if err != nil {
		return nil, err
	}

	assets := []*Asset{}
	assetIter := reg.db.IteratorPrefix([]byte(assetPrefix))
	defer assetIter.Release()

	for assetIter.Next() {
		asset := &Asset{}
		if err := json.Unmarshal(assetIter.Value(), asset); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}

	assetImage := &AssetImage{
		AssetIndex: assetIndex,
		Assets:     assets,
	}
	return assetImage, nil
}

func (reg *Registry) Restore(image *AssetImage) error {
	localIndex, err := reg.getNextAssetIndex()
	if err != nil {
		return err
	}

	if localIndex > image.AssetIndex {
		image.AssetIndex = localIndex
	}

	storeBatch := reg.db.NewBatch()
	for _, asset := range image.Assets {
		if existed := reg.db.Get(AliasKey(*asset.Alias)); existed != nil {
			return ErrDuplicateAlias
		}

		rawAsset, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		storeBatch.Set(AliasKey(*asset.Alias), []byte(asset.AssetID.String()))
		storeBatch.Set(Key(&asset.AssetID), rawAsset)
	}
	storeBatch.Write()
	reg.saveNextAssetIndex(image.AssetIndex)
	return nil
}
