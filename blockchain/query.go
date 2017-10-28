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

func (bcr *BlockchainReactor) listBalances(ctx context.Context, in requestQuery) interface{} {
	type assetAmount struct {
		AssetID string
		Amount  uint64
	}

	accountUTXOs := bcr.GetAccountUTXOs()
	accBalance := make(map[string]map[string]uint64)
	response := make([]string, 0)

	for _, accountUTXO := range accountUTXOs {

		assetID := fmt.Sprintf("%x", accountUTXO.AssetID)
		if _, ok := accBalance[accountUTXO.AccountID]; ok {
			if _, ok := accBalance[accountUTXO.AccountID][assetID]; ok {
				accBalance[accountUTXO.AccountID][assetID] += accountUTXO.Amount
			} else {
				accBalance[accountUTXO.AccountID][assetID] = accountUTXO.Amount
			}
		} else {
			accBalance[accountUTXO.AccountID] = map[string]uint64{assetID: accountUTXO.Amount}
		}
	}

	sortedAccount := []string{}
	for k, _ := range accBalance {
		sortedAccount = append(sortedAccount, k)
	}
	sort.Strings(sortedAccount)

	for _, account := range sortedAccount {
		sortedAsset := []string{}
		for k := range accBalance[account] {
			sortedAsset = append(sortedAsset, k)
		}
		sort.Strings(sortedAsset)

		assetAmounts := []assetAmount{}
		for _, asset := range sortedAsset {
			assetAmounts = append(assetAmounts, assetAmount{AssetID: asset, Amount: accBalance[account][asset]})
		}

		balanceString, _ := json.Marshal(assetAmounts)
		accBalancesString := fmt.Sprintf(`{"AccountID":"%s","Balances":"%s"}`, account, balanceString)
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
