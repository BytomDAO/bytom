package api

import (
	"context"
	"strings"

	"github.com/bytom/bytom/asset"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	chainjson "github.com/bytom/bytom/encoding/json"

	log "github.com/sirupsen/logrus"
)

// POST /create-asset
func (a *API) createAsset(ctx context.Context, ins struct {
	Alias           string                 `json:"alias"`
	RootXPubs       []chainkd.XPub         `json:"root_xpubs"`
	Quorum          int                    `json:"quorum"`
	Definition      map[string]interface{} `json:"definition"`
	LimitHeight     int64                  `json:"limit_height"`
	IssuanceProgram chainjson.HexBytes     `json:"issuance_program"`
}) Response {
	ass, err := a.wallet.AssetReg.Define(
		ins.RootXPubs,
		ins.Quorum,
		ins.Definition,
		ins.LimitHeight,
		strings.ToUpper(strings.TrimSpace(ins.Alias)),
		ins.IssuanceProgram,
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

// POST /update-asset-alias
func (a *API) updateAssetAlias(updateAlias struct {
	ID       string `json:"id"`
	NewAlias string `json:"alias"`
}) Response {
	if err := a.wallet.AssetReg.UpdateAssetAlias(updateAlias.ID, updateAlias.NewAlias); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(nil)
}
