package blockchain

import (
	"context"
	"sync"
	"time"

	"github.com/bytom/net/http/reqid"
	"github.com/bytom/log"
)

// POST /create-account-receiver
func (a *BlockchainReactor) createAccountReceiver(ctx context.Context, ins []struct {
	AccountID    string    `json:"account_id"`
	AccountAlias string    `json:"account_alias"`
	ExpiresAt    time.Time `json:"expires_at"`
}) []interface{} {
	log.Printf(ctx,"-------create-Account-Receiver-------")
	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			receiver, err := a.accounts.CreateReceiver(subctx, ins[i].AccountID, ins[i].AccountAlias, ins[i].ExpiresAt)
			if err != nil {
				responses[i] = err
			} else {
				responses[i] = receiver
			}
		}(i)
	}

	wg.Wait()
	return responses
}
