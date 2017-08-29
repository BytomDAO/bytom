package blockchain

import (
	"context"
	"sync"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/net/http/httpjson"
	"github.com/bytom/net/http/reqid"
	"github.com/bytom/log"
)

// POST /create-account
func (a *BlockchainReactor) createAccount(ctx context.Context, ins []struct {
	RootXPubs []chainkd.XPub `json:"root_xpubs"`
	Quorum    int
	Alias     string
	Tags      map[string]interface{}

	// ClientToken is the application's unique token for the account. Every account
	// should have a unique client token. The client token is used to ensure
	// idempotency of create account requests. Duplicate create account requests
	// with the same client_token will only create one account.
	ClientToken string `json:"client_token"`
}) interface{} {
	log.Printf(ctx, "-------createAccount-----")
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

	wg.Wait()
	return responses
}

// POST /update-account-tags
func (a *BlockchainReactor) updateAccountTags(ctx context.Context, ins []struct {
	ID    *string
	Alias *string
	Tags  map[string]interface{} `json:"tags"`
}) interface{} {
	log.Printf(ctx,"-------update-account-tags---------")
	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := range responses {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			//defer batchRecover(subctx, &responses[i])

			err := a.accounts.UpdateTags(subctx, ins[i].ID, ins[i].Alias, ins[i].Tags)
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
