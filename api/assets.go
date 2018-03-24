package api

import (
	"context"

	"github.com/bytom/asset"
	"github.com/bytom/crypto/ed25519/chainkd"

	log "github.com/sirupsen/logrus"
)

// POST /create-asset
func (a *API) createAsset(ctx context.Context, ins struct {
	Alias      string                 `json:"alias"`
	RootXPubs  []chainkd.XPub         `json:"root_xpubs"`
	Quorum     int                    `json:"quorum"`
	Definition map[string]interface{} `json:"definition"`
	Tags       map[string]interface{} `json:"tags"`
}) Response {
	ass, err := a.wallet.AssetReg.Define(
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
func (a *API) updateAssetTags(ctx context.Context, updateTag struct {
	AssetInfo string                 `json:"asset_info"`
	Tags      map[string]interface{} `json:"tags"`
}) Response {
	err := a.wallet.AssetReg.UpdateTags(nil, updateTag.AssetInfo, updateTag.Tags)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(nil)
}

// POST /update-asset-alias
func (a *API) updateAssetAlias(updateAlias struct {
	OldAlias string `json:"old_alias"`
	NewAlias string `json:"new_alias"`
}) Response {
	if err := a.wallet.AssetReg.UpdateAssetAlias(updateAlias.OldAlias, updateAlias.NewAlias); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(nil)
}
