package blockchain

import (
	"context"
//	"fmt"
//	"math"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/txfeed"
	"github.com/bytom/errors"
	"github.com/bytom/log"
	"github.com/bytom/net/http/httpjson"
)

// POST /create-txfeed
func (a *BlockchainReactor) createTxFeed(ctx context.Context, in struct {
	Alias  string
	Filter string

	// ClientToken is the application's unique token for the txfeed. Every txfeed
	// should have a unique client token. The client token is used to ensure
	// idempotency of create txfeed requests. Duplicate create txfeed requests
	// with the same client_token will only create one txfeed.
	ClientToken string `json:"client_token"`
}) (*txfeed.TxFeed, error) {
	log.Printf(ctx,"-------createTxFeed-------")
//	after := fmt.Sprintf("%d:%d-%d", a.chain.Height(), math.MaxInt32, uint64(math.MaxInt64))
//	return a.txFeeds.Create(ctx, in.Alias, in.Filter, after, in.ClientToken)
	return nil,nil
}

// POST /get-transaction-feed
func (a *BlockchainReactor) getTxFeed(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}) (*txfeed.TxFeed, error) {
	log.Printf(ctx,"-------getTxFeed-------")
//	return a.txFeeds.Find(ctx, in.ID, in.Alias)
	return nil,nil
}

// POST /delete-transaction-feed
func (a *BlockchainReactor) deleteTxFeed(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}) error {
	log.Printf(ctx,"-------deleteTxFeed-------")
//	return a.txFeeds.Delete(ctx, in.ID, in.Alias)
	return nil
}

// POST /update-transaction-feed
func (a *BlockchainReactor) updateTxFeed(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
	Prev  string `json:"previous_after"`
	After string `json:"after"`
}) (*txfeed.TxFeed, error) {
	log.Printf(ctx,"-------updateTxFeed-------")
	// TODO(tessr): Consider moving this function into the txfeed package.
	// (It's currently outside the txfeed package to avoid a dependecy cycle
	// between txfeed and query.)
	bad, err := txAfterIsBefore(in.After, in.Prev)
	if err != nil {
		return nil, err
	}

	if bad {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "new After cannot be before Prev")
	}
//	return a.txFeeds.Update(ctx, in.ID, in.Alias, in.After, in.Prev)
	return nil,nil
}

// txAfterIsBefore returns true if a is before b. It returns an error if either
// a or b are not valid query.TxAfters.
func txAfterIsBefore(a, b string) (bool, error) {
	aAfter, err := query.DecodeTxAfter(a)
	if err != nil {
		return false, err
	}

	bAfter, err := query.DecodeTxAfter(b)
	if err != nil {
		return false, err
	}

	return aAfter.FromBlockHeight < bAfter.FromBlockHeight ||
		(aAfter.FromBlockHeight == bAfter.FromBlockHeight &&
			aAfter.FromPosition < bAfter.FromPosition), nil
}

