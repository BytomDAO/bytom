package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/errors"
	"github.com/bytom/net/http/httpjson"
)

const (
	defGenericPageSize = 100
)

var accountUTXOFmt = `
{
	"OutputID":"%x",
	"AssetID":"%x",
	"Amount":"%d",
	"AccountID":"%s",
	"ProgramIndex":"%d",
	"Program":"%x",
	"SourceID":"%x",
	"SourcePos":"%d",
	"RefData":"%x",
	"Change":"%t"
}`

// POST /list-accounts
func (bcr *BlockchainReactor) listAccounts(ctx context.Context, query requestQuery) Response {

	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	accounts, after, last, err := bcr.accounts.ListAccounts(query.After, limit)
	if err != nil {
		log.Errorf("listAccounts: %v", err)
		return resWrapper(nil, err)
	}

	page := page{
		Items:    httpjson.Array(accounts),
		LastPage: last,
		After:     after}

	return resWrapper(page)
}

// POST /list-assets
func (bcr *BlockchainReactor) listAssets(ctx context.Context, query requestQuery) Response {

	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize // defGenericPageSize = 100
	}

	assets, after, last, err := bcr.assets.ListAssets(query.After, limit, defGenericPageSize)
	if err != nil {
		log.Errorf("listAssets: %v", err)
		return resWrapper(nil, err)
	}

	query.After = after

	page := &page{
		Items:    httpjson.Array(assets),
		LastPage: last,
		Next:     query}

	rawPage, err := json.Marshal(page)
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(rawPage)
}

// POST /listBalances
func (bcr *BlockchainReactor) listBalances(ctx context.Context, query requestQuery) Response {

	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize // defGenericPageSize = 100
	}

	accountUTXOs, _, _, err := bcr.wallet.GetAccountUTXOs("", 0, 0)
	if err != nil {
		log.Errorf("GetAccountUTXOs: %v", err)
		return resWrapper(nil, err)
	}

	balances, after, last, err := indexBalances(accountUTXOs, query.After, limit, defGenericPageSize)
	if err != nil {
		log.Errorf("listBalances: %v", err)
		return resWrapper(nil, err)
	}

	query.After = after

	page := &page{
		Items:    httpjson.Array(balances),
		LastPage: last,
		Next:     query}

	rawPage, err := json.Marshal(page)
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(rawPage)
}

func indexBalances(accountUTXOs []account.UTXO, after string, limit, defaultLimit int) ([]string, string, bool, error) {
	type assetAmount struct {
		AssetID string
		Amount  uint64
	}

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

	accBalance := make(map[string]map[string]uint64)
	balances := make([]string, 0)

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
		balances = append(balances, accBalancesString)
	}

	start, end := 0, len(balances)

	if len(balances) == 0 {
		return nil, "", true, errors.New("No account balances")
	} else if len(balances) > zafter {
		start = zafter
	} else {
		return nil, "", false, errors.WithDetailf(errors.New("Invalid after"), "value: %q", zafter)
	}

	if len(balances) > zafter+limit {
		end = zafter + limit
	}

	if len(balances) == end || len(balances) < defaultLimit {
		last = true
	}

	return balances[start:end], strconv.Itoa(end), last, nil
}

// POST /list-transactions
func (bcr *BlockchainReactor) listTransactions(ctx context.Context, query requestQuery) Response {
	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	transactions, after, last, err := bcr.wallet.GetTransactions(query.After, limit, defGenericPageSize)
	if err != nil {
		log.Errorf("listTransactions: %v", err)
		return resWrapper(nil, err)
	}

	query.After = after

	page := &page{
		Items:    httpjson.Array(transactions),
		LastPage: last,
		Next:     query}

	rawPage, err := json.Marshal(page)
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(rawPage)
}

// POST /list-unspent-outputs
func (bcr *BlockchainReactor) listUnspentOutputs(ctx context.Context, query requestQuery) Response {
	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize // defGenericPageSize = 100
	}

	accountUTXOs, after, last, err := bcr.wallet.GetAccountUTXOs(query.After, limit, defGenericPageSize)
	if err != nil {
		log.Errorf("list Unspent Outputs: %v", err)
		return resWrapper(nil, err)
	}

	formatUTXOs := make([]string, 0)
	for _, utxo := range accountUTXOs {
		format := fmt.Sprintf(accountUTXOFmt,
			utxo.OutputID, utxo.AssetID, utxo.Amount,
			utxo.AccountID, utxo.ProgramIndex, utxo.Program,
			utxo.SourceID, utxo.SourcePos, utxo.RefData, utxo.Change)

		formatUTXOs = append(formatUTXOs, format)
	}
	query.After = after

	page := &page{
		Items:    httpjson.Array(formatUTXOs),
		LastPage: last,
		Next:     query}

	rawPage, err := json.Marshal(page)
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(rawPage)

}
