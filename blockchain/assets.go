package blockchain

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/bytom/blockchain/asset"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/net/http/httpjson"
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
}) []byte {
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
	res, err := json.MarshalIndent(annotatedAsset, "", " ")
	if err != nil {
		return resWrapper(nil, err)
	}
	data := []string{string(res)}
	return resWrapper(data)
}

// POST /update-asset-tags
func (a *BlockchainReactor) updateAssetTags(ctx context.Context, ins []struct {
	ID    *string
	Alias *string
	Tags  map[string]interface{} `json:"tags"`
}) interface{} {
	log.Info("Update asset tags")
	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := range responses {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			err := a.assets.UpdateTags(subctx, ins[i].ID, ins[i].Alias, ins[i].Tags)
			if err != nil {
				responses[i] = err
			} else {
				responses[i] = httpjson.DefaultResponse
			}
		}(i)
	}

	wg.Wait()
	return responses
}
