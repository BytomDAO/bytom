package blockchain

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/txfeed"
)

// POST /create-txfeed
func (bcr *BlockchainReactor) createTxFeed(ctx context.Context, in struct {
	Alias  string
	Filter string
}) error {
	after := fmt.Sprintf("Height: %d", bcr.chain.Height())
	if err := bcr.txFeedTracker.Create(ctx, in.Alias, in.Filter, after); err != nil {
		log.WithField("error", err).Error("Add TxFeed Failed")
		return err
	}
	return nil
}

func (bcr *BlockchainReactor) getTxFeedByAlias(ctx context.Context, filter string) ([]*txfeed.TxFeed, error) {
	var (
		txFeed  = &txfeed.TxFeed{}
		txFeeds = []*txfeed.TxFeed{}
	)

	jf, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}

	value := bcr.txFeedTracker.DB.Get(jf)
	if value != nil {
		if err := json.Unmarshal(value, txFeed); err != nil {
			return nil, err
		}
		txFeeds = append(txFeeds, txFeed)
		return txFeeds, nil
	}
	return nil, nil
}

// POST /get-transaction-feed
func (bcr *BlockchainReactor) getTxFeed(ctx context.Context, in requestQuery) interface{} {
	txfeeds, err := bcr.getTxFeedByAlias(ctx, in.Filter)
	if err != nil {
		return err
	}
	return txfeeds

}

// POST /delete-transaction-feed
func (bcr *BlockchainReactor) deleteTxFeed(ctx context.Context, in struct {
	Alias string `json:"alias,omitempty"`
}) error {
	if err := bcr.txFeedTracker.Delete(ctx, in.Alias); err != nil {
		return err
	}
	return nil
}

// POST /update-transaction-feed
func (bcr *BlockchainReactor) updateTxFeed(ctx context.Context, in struct {
	Alias  string
	Filter string
}) error {
	if err := bcr.txFeedTracker.Delete(ctx, in.Alias); err != nil {
		return err
	}
	after := fmt.Sprintf("Height: %d", bcr.chain.Height())

	if err := bcr.txFeedTracker.Create(ctx, in.Alias, in.Filter, after); err != nil {
		return err
	}
	return nil
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

func (bcr *BlockchainReactor) getTxFeeds() ([]*txfeed.TxFeed, error) {
	var (
		txFeed  = &txfeed.TxFeed{}
		txFeeds = []*txfeed.TxFeed{}
	)
	iter := bcr.txFeedTracker.DB.Iterator()
	for iter.Next() {
		if err := json.Unmarshal(iter.Value(), &txFeed); err != nil {
			return nil, err
		}
		txFeeds = append(txFeeds, txFeed)
	}
	return txFeeds, nil
}

// listTxFeeds is an http handler for listing txfeeds. It does not take a filter.
// POST /list-transaction-feeds
func (bcr *BlockchainReactor) listTxFeeds(ctx context.Context, in requestQuery) interface{} {
	txfeeds, err := bcr.getTxFeeds()
	if err != nil {
		return err
	}
	return txfeeds
}
