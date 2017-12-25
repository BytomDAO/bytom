package blockchain

import (
	"context"
	"encoding/json"

	"github.com/bytom/blockchain/asset"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/net/http/reqid"

	log "github.com/sirupsen/logrus"
)

// POST /create-asset
func (a *BlockchainReactor) createAsset(ctx context.Context, ins struct {
	Alias      string
	RootXPubs  []chainkd.XPub `json:"root_xpubs"`
	Quorum     int
	Definition map[string]interface{}
	Tags       map[string]interface{}

	// ClientToken is the application's unique token for the asset. Every asset
	// should have a unique client token. The client token is used to ensure
	// idempotency of create asset requests. Duplicate create asset requests
	// with the same client_token will only create one asset.
	AccessToken string `json:"access_token"`
}) Response {
	subctx := reqid.NewSubContext(ctx, reqid.New())

	ass, err := a.assets.Define(
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

	log.WithField("asset", annotatedAsset).Info("Created asset")

	return resWrapper(annotatedAsset)
}

// POST /update-asset-tags
func (a *BlockchainReactor) updateAssetTags(ctx context.Context, updateTag struct {
	AssetInfo string
	Tags      map[string]interface{} `json:"tags"`
}) Response {

	log.Info("%v", updateTag)
	err := a.assets.UpdateTags(nil, updateTag.AssetInfo, updateTag.Tags)
	if err != nil {
		resWrapper(nil, err)
	}

	return resWrapper(nil)
}
