package blockchain

import (
	"context"

	"github.com/bytom/blockchain/asset"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/net/http/reqid"

	log "github.com/sirupsen/logrus"
)

// POST /create-asset
func (bcr *BlockchainReactor) createAsset(ctx context.Context, ins struct {
	Alias      string                 `json:"alias"`
	RootXPubs  []chainkd.XPub         `json:"root_xpubs"`
	Quorum     int                    `json:"quorum"`
	Definition map[string]interface{} `json:"definition"`
	Tags       map[string]interface{} `json:"tags"`

	// ClientToken is the application's unique token for the asset. Every asset
	// should have a unique client token. The client token is used to ensure
	// idempotency of create asset requests. Duplicate create asset requests
	// with the same client_token will only create one asset.
	AccessToken string `json:"access_token"`
}) Response {
	subctx := reqid.NewSubContext(ctx, reqid.New())

	ass, err := bcr.assets.Define(
		subctx,
		ins.RootXPubs,
		ins.Quorum,
		ins.Definition,
		ins.Alias,
		ins.Tags,
		ins.AccessToken,
	)
	if err != nil {
		return resWrapper(nil, err)
	}

	annotatedAsset, err := asset.Annotated(ass)
	if err != nil {
		return resWrapper(nil, err)
	}

	log.WithField("asset ID", annotatedAsset.ID.String()).Info("Created asset")

	return resWrapper(annotatedAsset)
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
