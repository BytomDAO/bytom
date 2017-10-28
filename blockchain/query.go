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
		"AccountID":"%s","ProgramIndex":"%d","Program":"%x",
		"BlockHeight":"%d","SourceID":"%x","SourcePos":"%d",
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

	iter := bcr.pinStore.DB.IteratorPrefix([]byte("acu"))
	for iter.Next() {

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
		Amount  uint64
	}

	var (
		aaTmp       = assetAmount{}
		accBalances = make(map[string][]assetAmount)
		keys        = make([]string, 0)
		response    = make([]string, 0)
		exist       = false
		index       = 0
	)

	accountUTXOs := bcr.GetAccountUTXOs()

	for _, res := range accountUTXOs {

		aaTmp.AssetID = fmt.Sprintf("%x", res.AssetID)
		aaTmp.Amount = res.Amount
		if _, ok := accBalances[res.AccountID]; ok {

			for i, aA := range accBalances[res.AccountID] {
				if aA.AssetID == aaTmp.AssetID {
					exist = true
					index = i
					break
				}
			}

			if exist {
				accBalances[res.AccountID][index].Amount += aaTmp.Amount
				exist = false
			} else {
				accBalances[res.AccountID] = append(accBalances[res.AccountID], aaTmp)
			}

		} else {
			accBalances[res.AccountID] = append(accBalances[res.AccountID], aaTmp)
		}
	}

	for k := range accBalances {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		balanceString, _ := json.Marshal(accBalances[k])
		accBalancesString := fmt.Sprintf(`{"AccountID":"%s","Balances":"%s"}`, k, balanceString)
		response = append(response, accBalancesString)
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
			res.AccountID, res.ProgramIndex, res.Program,
			res.BlockHeight, res.SourceID, res.SourcePos,
			res.RefData, res.Change)

		response = append(response, restring)
	}

	return response
}
