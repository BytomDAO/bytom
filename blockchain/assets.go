package blockchain

import (
	"context"

	"github.com/bytom/blockchain/asset"
	"github.com/bytom/crypto/ed25519/chainkd"

	log "github.com/sirupsen/logrus"
)

// POST /create-asset
func (bcr *BlockchainReactor) createAsset(ctx context.Context, ins struct {
	Alias      string                 `json:"alias"`
	RootXPubs  []chainkd.XPub         `json:"root_xpubs"`
	Quorum     int                    `json:"quorum"`
	Definition map[string]interface{} `json:"definition"`
	Tags       map[string]interface{} `json:"tags"`
}) Response {
	ass, err := bcr.assets.Define(
		ins.RootXPubs,
		ins.Quorum,
		ins.Definition,
		ins.Alias,
		ins.Tags,
	)
	if err != nil {
		return NewErrorResponse(err)
	}

	annotatedAsset, err := asset.Annotated(ass)
	if err != nil {
		return NewErrorResponse(err)
	}

	log.WithField("asset ID", annotatedAsset.ID.String()).Info("Created asset")

	return NewSuccessResponse(annotatedAsset)
}

// POST /update-asset-tags
func (bcr *BlockchainReactor) updateAssetTags(ctx context.Context, updateTag struct {
	AssetInfo string                 `json:"asset_info"`
	Tags      map[string]interface{} `json:"tags"`
}) Response {
	err := bcr.assets.UpdateTags(nil, updateTag.AssetInfo, updateTag.Tags)
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(nil)
}

// POST /rename-asset-alias
func (bcr *BlockchainReactor) updateAssetAlias(updateAlias struct {
	OldAlias string `json:"old_alias"`
	NewAlias string `json:"new_alias"`
}) Response {
	err := bcr.assets.UpdateAssetAlias(updateAlias.OldAlias, updateAlias.NewAlias)
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(nil)
}
