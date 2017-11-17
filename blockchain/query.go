package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/errors"
	"github.com/bytom/net/http/httpjson"
)

const (
	defGenericPageSize = 100
)

var (
	accountUTXOFmt = `
	{
		"OutputID":"%x","AssetID":"%x","Amount":"%d",
		"AccountID":"%s","ProgramIndex":"%d","Program":"%x",
		"SourceID":"%x","SourcePos":"%d","RefData":"%x","Change":"%t"
	}`
)

//
// POST /list-accounts
func (bcr *BlockchainReactor) listAccounts(ctx context.Context, in requestQuery) interface{} {
	response, err := bcr.accounts.QueryAll(ctx)
	if err != nil {
		log.Errorf("listAccounts: %v", err)
	}
	return response

}

//
// POST /list-assets
func (bcr *BlockchainReactor) listAssets(ctx context.Context, in requestQuery) interface{} {

	response, _ := bcr.assets.QueryAll(ctx)

	return response
}

//GetAccountUTXOs return all account unspent outputs
func (bcr *BlockchainReactor) GetAccountUTXOs() []account.UTXO {

	var (
		accountUTXO  = account.UTXO{}
		accountUTXOs = make([]account.UTXO, 0)
	)

	accountUTXOIter := bcr.wallet.DB.IteratorPrefix([]byte(account.UTXOPreFix))
	defer accountUTXOIter.Release()
	for accountUTXOIter.Next() {

		if err := json.Unmarshal(accountUTXOIter.Value(), &accountUTXO); err != nil {
			hashKey := accountUTXOIter.Key()[len(account.UTXOPreFix):]
			log.WithField("UTXO hash", string(hashKey)).Warn("get account UTXO")
			continue
		}

		accountUTXOs = append(accountUTXOs, accountUTXO)
	}

	return accountUTXOs
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
	for k := range accBalance {
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

// POST /list-unspent-outputs
func (bcr *BlockchainReactor) listUnspentOutputs(ctx context.Context, in requestQuery) interface{} {

	var (
		response = make([]string, 0)
		restring = ""
	)

	accoutUTXOs := bcr.GetAccountUTXOs()

	for _, res := range accoutUTXOs {

		restring = fmt.Sprintf(accountUTXOFmt,
			res.OutputID, res.AssetID, res.Amount,
			res.AccountID, res.ProgramIndex, res.Program,
			res.SourceID, res.SourcePos, res.RefData, res.Change)

		response = append(response, restring)
	}

	return response
}
