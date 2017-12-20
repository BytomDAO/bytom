package blockchain

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/net/http/httpjson"
	"github.com/bytom/net/http/reqid"

	log "github.com/sirupsen/logrus"
)

// POST /create-account
func (a *BlockchainReactor) createAccount(ctx context.Context, ins struct {
	RootXPubs []chainkd.XPub `json:"root_xpubs"`
	Quorum    int
	Alias     string
	Tags      map[string]interface{}

	// ClientToken is the application's unique token for the account. Every account
	// should have a unique client token. The client token is used to ensure
	// idempotency of create account requests. Duplicate create account requests
	// with the same client_token will only create one account.
	ClientToken string `json:"client_token"`
}) []byte {
	type resCreateAccount struct {
		Account *query.AnnotatedAccount `json:"account"`
	}
	acc, err := a.accounts.Create(nil, ins.RootXPubs, ins.Quorum, ins.Alias, ins.Tags, ins.ClientToken)
	if err != nil {
		return resWrapper(nil, err)
	}
	annotatedAccount, err := account.Annotated(acc)
	if err != nil {
		return resWrapper(nil, err)
	}
	log.WithField("account", annotatedAccount).Info("Created account")
	resAccount := &resCreateAccount{annotatedAccount}
	res, err := json.Marshal(annotatedAccount)
	if err != nil {
		return resWrapper(nil, err)
	}
	data := []string{string(res)}
	return resWrapper(data)
}

// POST /update-account-tags
func (a *BlockchainReactor) updateAccountTags(ctx context.Context, ins []struct {
	ID    *string
	Alias *string
	Tags  map[string]interface{} `json:"tags"`
}) interface{} {
	log.Info("Updating account tags")
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
