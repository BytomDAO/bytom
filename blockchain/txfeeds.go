package blockchain

import (
	"context"
	"encoding/json"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/errors"
	"github.com/bytom/net/http/httpjson"
)

// POST /create-txfeed
func (bcr *BlockchainReactor) createTxFeed(ctx context.Context, in struct {
	Alias  string
	Filter string
}) Response {
	if err := bcr.txFeedTracker.Create(ctx, in.Alias, in.Filter); err != nil {
		log.WithField("error", err).Error("Add TxFeed Failed")
		return resWrapper(nil, err)
	}
	return resWrapper(nil)
}

func (bcr *BlockchainReactor) getTxFeedByAlias(ctx context.Context, filter string) ([]byte, error) {
	jf, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}

	value := bcr.txFeedTracker.DB.Get(jf)
	if value == nil {
		return nil, errors.New("No transaction feed")
	}

	return value, nil
}

// POST /get-transaction-feed
func (bcr *BlockchainReactor) getTxFeed(ctx context.Context, in struct {
	Alias string `json:"alias,omitempty"`
}) Response {
	txfeed, err := bcr.getTxFeedByAlias(ctx, in.Alias)
	if err != nil {
		return resWrapper(nil, err)
	}
	return resWrapper(txfeed)
}

// POST /delete-transaction-feed
func (bcr *BlockchainReactor) deleteTxFeed(ctx context.Context, in struct {
	Alias string `json:"alias,omitempty"`
}) Response {
	if err := bcr.txFeedTracker.Delete(ctx, in.Alias); err != nil {
		return resWrapper(nil, err)
	}
	return resWrapper(nil)
}

// POST /update-transaction-feed
func (bcr *BlockchainReactor) updateTxFeed(ctx context.Context, in struct {
	Alias  string
	Filter string
}) Response {
	if err := bcr.txFeedTracker.Delete(ctx, in.Alias); err != nil {
		return resWrapper(nil, err)
	}
	if err := bcr.txFeedTracker.Create(ctx, in.Alias, in.Filter); err != nil {
		log.WithField("error", err).Error("Update TxFeed Failed")
		return resWrapper(nil, err)
	}
	return resWrapper(nil)
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

func (bcr *BlockchainReactor) getTxFeeds(after string, limit, defaultLimit int) ([]string, string, bool, error) {
	var (
		zafter int
		err    error
		last   bool
	)

	if after != "" {
		zafter, err = strconv.Atoi(after)
		if err != nil {
			return nil, "", false, errors.WithDetailf(errors.New("Invalid after"), "value: %q", zafter)
		}
	}

	txFeeds := make([]string, 0)
	iter := bcr.txFeedTracker.DB.Iterator()
	defer iter.Release()

	for iter.Next() {
		txFeeds = append(txFeeds, string(iter.Value()))
	}

	start, end := 0, len(txFeeds)

	if len(txFeeds) == 0 {
		return nil, "", true, errors.New("No transaction feed")
	} else if len(txFeeds) > zafter {
		start = zafter
	} else {
		return nil, "", false, errors.WithDetailf(errors.New("Invalid after"), "value: %v", zafter)
	}
	if len(txFeeds) > zafter+limit {
		end = zafter + limit
	}

	if len(txFeeds) == end || len(txFeeds) < defaultLimit {
		last = true
	}

	return txFeeds[start:end], strconv.Itoa(end), last, nil
}

// listTxFeeds is an http handler for listing txfeeds. It does not take a filter.
// POST /list-transaction-feeds
func (bcr *BlockchainReactor) listTxFeeds(ctx context.Context, query requestQuery) Response {
	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	txfeeds, after, last, err := bcr.getTxFeeds(query.After, limit, defGenericPageSize)
	if err != nil {
		return resWrapper(nil, err)
	}

	var items []string
	for _, txfeed := range txfeeds {
		items = append(items, txfeed)
	}

	query.After = after
	page := &page{
		Items:    httpjson.Array(items),
		LastPage: last,
		Next:     query}

	rawPage, err := json.Marshal(page)
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(rawPage)
}
