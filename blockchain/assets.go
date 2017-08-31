package blockchain

import (
	"context"
	"sync"

//	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/account"
	"github.com/bytom/crypto/ed25519/chainkd"
//	"github.com/bytom/net/http/httpjson"
	"github.com/bytom/net/http/reqid"
	"github.com/bytom/log"
)

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
	log.Printf(ctx,"------createAsset-----")
	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := range responses {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			//defer batchRecover(subctx, &responses[i])

			acc, err := a.accounts.Create(subctx, ins[i].RootXPubs, ins[i].Quorum, ins[i].Alias, ins[i].Tags, ins[i].ClientToken)
			if err != nil {
				responses[i] = err
				return
			}
			aa, err := account.Annotated(acc)
			if err != nil {
				responses[i] = err
				return
			}
			responses[i] = aa
		}(i)
	}

//	wg.wait()
	return responses,nil
}
