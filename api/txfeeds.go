package api

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/txfeed"
	"github.com/bytom/errors"
)

// POST /create-txfeed
func (a *API) createTxFeed(ctx context.Context, in struct {
	Alias  string `json:"alias"`
	Filter string `json:"filter"`
}) Response {
	if err := a.txFeedTracker.Create(ctx, in.Alias, in.Filter); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Error("Add TxFeed Failed")
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(nil)
}

// POST /get-transaction-feed
func (a *API) getTxFeed(ctx context.Context, in struct {
	Alias string `json:"alias,omitempty"`
}) Response {
	var tmpTxFeed interface{}
	rawTxfeed, err := a.GetTxFeedByAlias(ctx, in.Alias)
	if err != nil {
		return NewErrorResponse(err)
	}
	err = json.Unmarshal(rawTxfeed, &tmpTxFeed)
	if err != nil {
		return NewErrorResponse(err)
	}
	data := map[string]interface{}{"txfeed": tmpTxFeed}
	return NewSuccessResponse(data)
}

// POST /delete-transaction-feed
func (a *API) deleteTxFeed(ctx context.Context, in struct {
	Alias string `json:"alias,omitempty"`
}) Response {
	if err := a.txFeedTracker.Delete(ctx, in.Alias); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(nil)
}

// POST /update-transaction-feed
func (a *API) updateTxFeed(ctx context.Context, in struct {
	Alias  string `json:"alias"`
	Filter string `json:"filter"`
}) Response {
	if err := a.txFeedTracker.Delete(ctx, in.Alias); err != nil {
		return NewErrorResponse(err)
	}
	if err := a.txFeedTracker.Create(ctx, in.Alias, in.Filter); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Error("Update TxFeed Failed")
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(nil)
}

func (a *API) getTxFeeds() ([]txfeed.TxFeed, error) {
	txFeed := txfeed.TxFeed{}
	txFeeds := make([]txfeed.TxFeed, 0)

	iter := a.txFeedTracker.DB.Iterator()
	defer iter.Release()

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
func (a *API) listTxFeeds(ctx context.Context) Response {
	txFeeds, err := a.getTxFeeds()
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(txFeeds)
}

func (a *API) GetTxFeedByAlias(ctx context.Context, filter string) ([]byte, error) {
	jf, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}

	value := a.txFeedTracker.DB.Get(jf)
	if value == nil {
		return nil, errors.New("No transaction feed")
	}

	return value, nil
}
