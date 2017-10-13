package blockchain

import (
	"context"
	"math"

	"github.com/bytom/blockchain/query"
	//"github.com/bytom/blockchain/query/filter"
	"github.com/bytom/errors"
	"github.com/bytom/net/http/httpjson"
	//"github.com/bytom/log"
	"encoding/json"
	"github.com/bytom/blockchain/account"
	"fmt"
	"sort"
)

const (
	defGenericPageSize = 100
)
var (
	AccountUtxoFmt = `
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

func (bcr *BlockchainReactor) Getaccountutxos()([]account.AccountUtxos){

	var (
		au = account.AccountUtxos{}
		accutoxs = []account.AccountUtxos{}
	)

	iter := bcr.pinStore.DB.Iterator()
	for iter.Next() {
		key := string(iter.Key())
		if key[:3] != "acu" {
			continue
		}

		err := json.Unmarshal(iter.Value(),&au)
		if err != nil {
			continue
		}

		accutoxs = append(accutoxs, au)
	}

	return accutoxs
}

// POST /list-balances
func (bcr *BlockchainReactor) listBalances(ctx context.Context, in requestQuery) interface{} {

	type assetmount struct {
		AssetID string
		Amount int64
	}
	var (
		am	= assetmount{}
		accbalances = make(map[string][]assetmount,0)
		accbalancesSort = make(map[string][]assetmount,0)
		keys = make([]string,0)
		response = make([]interface{},0)
	)

	accoututxos := bcr.Getaccountutxos()

	for _,res := range accoututxos{

		am.AssetID = fmt.Sprintf("%x",res.AssetID)
		am.Amount  = res.Amount
		if _, ok := accbalances[res.AccountID]; ok {
			for _,amentry := range accbalances[res.AccountID]{
				if amentry.AssetID == am.AssetID {
					amentry.Amount += am.Amount
				}else{
					accbalances[res.AccountID]=append(accbalances[res.AccountID],am)
				}
			}
		}else{
			accbalances[res.AccountID]=append(accbalances[res.AccountID],am)
		}

	}

	for k := range accbalances {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		accbalancesSort[k] = accbalances[k]
	}

	if len(accbalancesSort)!=0 {
		response = append(response,accbalancesSort)
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
func (bcr *BlockchainReactor) listUnspentOutputs(ctx context.Context, in requestQuery) interface{}  {

	var (
		response = make([]string,0)
		restring = ""
	)

	accoututxos := bcr.Getaccountutxos()

	for _,res := range accoututxos{

		restring = fmt.Sprintf(AccountUtxoFmt,
			res.OutputID,res.AssetID,res.Amount,
			res.AccountID,res.CpIndex,res.Program,
			res.Confirmed,res.SourceID,res.SourcePos,
			res.RefData,res.Change)

		response = append(response,restring)
	}

	return response
}
