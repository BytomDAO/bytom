package asset

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/protocol/bc"
)

// AnnotateTxs adds asset data to transactions
func (reg *Registry) AnnotateTxs(txs []*query.AnnotatedTx) error {

	assetIDMap := make(map[bc.AssetID]bool)

	// Collect all of the asset IDs appearing in the entire block. We only
	// check the outputs because every transaction should balance.
	for _, tx := range txs {
		for _, out := range tx.Outputs {
			assetIDMap[out.AssetID] = true
		}
	}
	if len(assetIDMap) == 0 {
		return nil
	}

	// Look up all the asset tags for all applicable assets.
	asset := Asset{}
	tagsByAssetID := make(map[bc.AssetID]*json.RawMessage)
	defsByAssetID := make(map[bc.AssetID]*json.RawMessage)
	aliasesByAssetID := make(map[bc.AssetID]string)
	localByAssetID := make(map[bc.AssetID]bool)

	for assetID := range assetIDMap {
		rawAsset := reg.db.Get([]byte(assetID.String()))
		if rawAsset == nil {
			//local no asset
			continue
		}

		if err := json.Unmarshal(rawAsset, &asset); err != nil {
			log.WithFields(log.Fields{"warn": err, "asset id": assetID.String()}).Warn("look up asset")
			continue
		}

		annotatedAsset, err := Annotated(&asset)
		if err != nil {
			log.WithFields(log.Fields{"warn": err, "asset id": assetID.String()}).Warn("annotated asset")
			continue
		}

		if annotatedAsset.Alias != "" {
			aliasesByAssetID[assetID] = annotatedAsset.Alias
		}

		localByAssetID[assetID] = annotatedAsset.IsLocal == true

		if annotatedAsset.Tags != nil {
			tagsByAssetID[assetID] = annotatedAsset.Tags
		}

		if annotatedAsset.Definition != nil {
			defsByAssetID[assetID] = annotatedAsset.Definition
		}

	}

	empty := json.RawMessage(`{}`)
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if alias, ok := aliasesByAssetID[in.AssetID]; ok {
				in.AssetAlias = alias
			}
			if localByAssetID[in.AssetID] {
				in.AssetIsLocal = true
			}

			in.AssetTags = &empty
			in.AssetDefinition = &empty
			if tags := tagsByAssetID[in.AssetID]; tags != nil {
				in.AssetTags = tags
			}
			if def := defsByAssetID[in.AssetID]; def != nil {
				in.AssetDefinition = def
			}
		}

		for _, out := range tx.Outputs {
			if alias, ok := aliasesByAssetID[out.AssetID]; ok {
				out.AssetAlias = alias
			}
			if localByAssetID[out.AssetID] {
				out.AssetIsLocal = true
			}

			out.AssetTags = &empty
			out.AssetDefinition = &empty
			if tags := tagsByAssetID[out.AssetID]; tags != nil {
				out.AssetTags = tags
			}
			if def := defsByAssetID[out.AssetID]; def != nil {
				out.AssetDefinition = def
			}
		}
	}

	return nil
}
