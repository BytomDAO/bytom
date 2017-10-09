package blockchain

import (
	"context"
	"sync"

	"github.com/bytom/blockchain/asset"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/log"
	"github.com/bytom/net/http/httpjson"
	"github.com/bytom/net/http/reqid"
)

// POST /create-asset
func (a *BlockchainReactor) createAsset(ctx context.Context, ins []struct {
	Alias      string
	RootXPubs  []chainkd.XPub `json:"root_xpubs"`
	Quorum     int
	Definition map[string]interface{}
	Tags       map[string]interface{}

	// ClientToken is the application's unique token for the asset. Every asset
	// should have a unique client token. The client token is used to ensure
	// idempotency of create asset requests. Duplicate create asset requests
	// with the same client_token will only create one asset.
	ClientToken string `json:"client_token"`
}) ([]interface{}, error) {
	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := range responses {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			a, err := a.assets.Define(
				subctx,
				ins[i].RootXPubs,
				ins[i].Quorum,
				ins[i].Definition,
				ins[i].Alias,
				ins[i].Tags,
				ins[i].ClientToken,
			)
			if err != nil {
				responses[i] = err
				return
			}
			aa, err := asset.Annotated(a)
			log.Printf(ctx, "-------createAsset-----Annotated asset:%v", aa)
			if err != nil {
				responses[i] = err
				return
			}
			responses[i] = aa
		}(i)
	}

	wg.Wait()
	log.Printf(ctx, "-------createAsset-----%v", responses)
	return responses, nil
}

// POST /update-asset-tags
func (a *BlockchainReactor) updateAssetTags(ctx context.Context, ins []struct {
	ID    *string
	Alias *string
	Tags  map[string]interface{} `json:"tags"`
}) interface{} {
	log.Printf(ctx, "------updateAssetTags-----")
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
