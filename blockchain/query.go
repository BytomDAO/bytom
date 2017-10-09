package blockchain

import (
	"context"
	"math"

	"github.com/bytom/blockchain/query"
	//"github.com/bytom/blockchain/query/filter"
	"github.com/bytom/errors"
	"github.com/bytom/net/http/httpjson"
	//"github.com/bytom/log"
)

const (
	defGenericPageSize = 100
)

//
// POST /list-accounts
func (bcr *BlockchainReactor) listAccounts(ctx context.Context, in requestQuery) interface{} {

	response, _ := bcr.accounts.QueryAll(ctx)

	return response

}

//
// POST /list-assets
func (bcr *BlockchainReactor) listAssets(ctx context.Context, in requestQuery) interface{} {

	response, _ := bcr.assets.QueryAll(ctx)

	return response
}

// POST /list-balances
func (bcr *BlockchainReactor) listBalances(ctx context.Context, in requestQuery) interface{} {

	response := bcr.chain.GetAssetsAmount()
	if len(response) == 0 {
		return nil
	} else {
		return response
	}

}

// listTransactions is an http handler for listing transactions matching
// an index or an ad-hoc filter.
//
// POST /list-transactions
func (bcr *BlockchainReactor) listTransactions(ctx context.Context, in requestQuery) (result page, err error) {
	var c context.CancelFunc
	timeout := in.Timeout.Duration
	if timeout != 0 {
		ctx, c = context.WithTimeout(ctx, timeout)
		defer c()
	}

	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	endTimeMS := in.EndTimeMS
	if endTimeMS == 0 {
		endTimeMS = math.MaxInt64
	} else if endTimeMS > math.MaxInt64 {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "end timestamp is too large")
	}

	// Either parse the provided `after` or look one up for the time range.
	//	var after query.TxAfter
	if in.After != "" {
		_, err = query.DecodeTxAfter(in.After)
		if err != nil {
			return result, errors.Wrap(err, "decoding `after`")
		}
	} else {
		/*		after, err = bcr.indexer.LookupTxAfter(ctx, in.StartTimeMS, endTimeMS)
				if err != nil {
					return result, err
				}
		*/
	}

	/*	txns, nextAfter, err := bcr.indexer.Transactions(ctx, in.Filter, in.FilterParams, after, limit, in.AscLongPoll)
		if err != nil {
			return result, errors.Wrap(err, "running tx query")
		}
	*/
	out := in
	//	out.After = nextAfter.String()
	return page{
		//		Items:    httpjson.Array(txns),
		//		LastPage: len(txns) < limit,
		Next: out,
	}, nil
}

// listTxFeeds is an http handler for listing txfeeds. It does not take a filter.
//
// POST /list-transaction-feeds
func (bcr *BlockchainReactor) listTxFeeds(ctx context.Context, in requestQuery) (page, error) {
	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	after := in.After

	/*	txfeeds, after, err := bcr.txFeeds.Query(ctx, after, limit)
		if err != nil {
			return page{}, errors.Wrap(err, "running txfeed query")
		}
	*/
	out := in
	out.After = after
	return page{
		//		Items:    httpjson.Array(txfeeds),
		//		LastPage: len(txfeeds) < limit,
		Next: out,
	}, nil
}

// POST /list-unspent-outputs
func (bcr *BlockchainReactor) listUnspentOutputs(ctx context.Context, in requestQuery) (result page, err error) {
	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	//	var after *query.OutputsAfter
	if in.After != "" {
		_, err = query.DecodeOutputsAfter(in.After)
		if err != nil {
			return result, errors.Wrap(err, "decoding `after`")
		}
	}

	timestampMS := in.TimestampMS
	if timestampMS == 0 {
		timestampMS = math.MaxInt64
	} else if timestampMS > math.MaxInt64 {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "timestamp is too large")
	}
	/*	outputs, nextAfter, err := bcr.indexer.Outputs(ctx, in.Filter, in.FilterParams, timestampMS, after, limit)
		if err != nil {
			return result, errors.Wrap(err, "querying outputs")
		}
	*/
	outQuery := in
	//	outQuery.After = nextAfter.String()
	return page{
		//		Items:    httpjson.Array(outputs),
		//		LastPage: len(outputs) < limit,
		Next: outQuery,
	}, nil
}
