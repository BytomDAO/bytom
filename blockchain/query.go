package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/errors"
	"github.com/bytom/net/http/httpjson"
)

const (
	defGenericPageSize = 100
)

var (
	AccountUTXOFmt = `
	{
		"OutputID":"%x","AssetID":"%x","Amount":"%d",
		"AccountID":"%s","CpIndex":"%d","Program":"%x",
		"Confirmed":"%d","SourceID":"%x","SourcePos":"%d",
		"RefData":"%x","Change":"%t"
	}`
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

func (bcr *BlockchainReactor) GetAccountUTXOs() []account.AccountUTXOs {

	var (
		au       = account.AccountUTXOs{}
		accutoxs = []account.AccountUTXOs{}
	)

	iter := bcr.pinStore.DB.Iterator()
	for iter.Next() {
		key := string(iter.Key())
		if key[:3] != "acu" {
			continue
		}

		err := json.Unmarshal(iter.Value(), &au)
		if err != nil {
			continue
		}

		accutoxs = append(accutoxs, au)
	}

	return accutoxs
}

// POST /list-balances
func (bcr *BlockchainReactor) listBalances(ctx context.Context, in requestQuery) interface{} {

	type assetAmount struct {
		AssetID string
		Amount  int64
	}
	var (
		aa              = assetAmount{}
		accBalances     = make(map[string][]assetAmount, 0)
		accBalancesSort = make(map[string][]assetAmount, 0)
		keys            = make([]string, 0)
		response        = make([]interface{}, 0)
	)

	accoutUTXOs := bcr.GetAccountUTXOs()

	for _, res := range accoutUTXOs {

		aa.AssetID = fmt.Sprintf("%x", res.AssetID)
		aa.Amount = res.Amount
		if _, ok := accBalances[res.AccountID]; ok {
			for _, amentry := range accBalances[res.AccountID] {
				if amentry.AssetID == aa.AssetID {
					amentry.Amount += aa.Amount
				} else {
					accBalances[res.AccountID] = append(accBalances[res.AccountID], aa)
				}
			}
		} else {
			accBalances[res.AccountID] = append(accBalances[res.AccountID], aa)
		}

	}

	for k := range accBalances {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		accBalancesSort[k] = accBalances[k]
	}

	if len(accBalancesSort) != 0 {
		response = append(response, accBalancesSort)
	}

	return response
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
func (bcr *BlockchainReactor) listUnspentOutputs(ctx context.Context, in requestQuery) interface{} {

	var (
		response = make([]string, 0)
		restring = ""
	)

	accoutUTXOs := bcr.GetAccountUTXOs()

	for _, res := range accoutUTXOs {

		restring = fmt.Sprintf(AccountUTXOFmt,
			res.OutputID, res.AssetID, res.Amount,
			res.AccountID, res.CpIndex, res.Program,
			res.Confirmed, res.SourceID, res.SourcePos,
			res.RefData, res.Change)

		response = append(response, restring)
	}

	return response
}
