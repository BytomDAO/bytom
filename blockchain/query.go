package blockchain

import (
	"context"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/account"
)

// POST /list-accounts
func (bcr *BlockchainReactor) listAccounts(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	accounts, err := bcr.accounts.ListAccounts(filter.ID)
	if err != nil {
		log.Errorf("listAccounts: %v", err)
		return resWrapper(nil, err)
	}

	return resWrapper(accounts)
}

// POST /list-assets
func (bcr *BlockchainReactor) listAssets(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	assets, err := bcr.assets.ListAssets(filter.ID)
	if err != nil {
		log.Errorf("listAssets: %v", err)
		return resWrapper(nil, err)
	}

	return resWrapper(assets)
}

// POST /listBalances
func (bcr *BlockchainReactor) listBalances(ctx context.Context) Response {
	accountUTXOs, err := bcr.wallet.GetAccountUTXOs("")
	if err != nil {
		log.Errorf("GetAccountUTXOs: %v", err)
		return resWrapper(nil, err)
	}

	return resWrapper(bcr.indexBalances(accountUTXOs))
}

type assetAmount struct {
	AssetID string `json:"asset_id"`
	Amount  uint64 `json:"amount"`
}

type accountBalance struct {
	AccountID string        `json:"id"`
	Alias     string        `json:"alias,omitempty"`
	Balances  []assetAmount `json:"balances"`
}

func (bcr *BlockchainReactor) indexBalances(accountUTXOs []account.UTXO) []accountBalance {
	accBalance := make(map[string]map[string]uint64)
	balances := make([]accountBalance, 0)
	tmpBalance := accountBalance{}

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

	for _, id := range sortedAccount {
		sortedAsset := []string{}
		for k := range accBalance[id] {
			sortedAsset = append(sortedAsset, k)
		}
		sort.Strings(sortedAsset)

		assetAmounts := []assetAmount{}
		for _, asset := range sortedAsset {
			assetAmounts = append(assetAmounts, assetAmount{AssetID: asset, Amount: accBalance[id][asset]})
		}

		alias := bcr.accounts.GetAliasByID(id)

		tmpBalance.Alias = alias
		tmpBalance.AccountID = id
		tmpBalance.Balances = assetAmounts

		balances = append(balances, tmpBalance)
	}

	return balances
}

// POST /list-transactions
func (bcr *BlockchainReactor) listTransactions(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	transactions, err := bcr.wallet.GetTransactions(filter.ID)
	if err != nil {
		log.Errorf("listTransactions: %v", err)
		return resWrapper(nil, err)
	}

	return resWrapper(transactions)
}

type annotatedUTXO struct {
	Alias        string `json:"alias,omitempty"`
	OutputID     string `json:"id"`
	AssetID      string `json:"asset_id"`
	Amount       uint64 `json:"amount"`
	AccountID    string `json:"account_id"`
	ProgramIndex uint64 `json:"program_index"`
	Program      string `json:"program"`
	SourceID     string `json:"source_id"`
	SourcePos    uint64 `json:"source_pos"`
	RefData      string `json:"ref_data"`
	Change       bool   `json:"change"`
}

// POST /list-unspent-outputs
func (bcr *BlockchainReactor) listUnspentOutputs(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	tmpUTXO := annotatedUTXO{}
	UTXOs := make([]annotatedUTXO, 0)

	accountUTXOs, err := bcr.wallet.GetAccountUTXOs(filter.ID)
	if err != nil {
		log.Errorf("list Unspent Outputs: %v", err)
		return resWrapper(nil, err)
	}

	for _, utxo := range accountUTXOs {
		tmpUTXO.Alias = bcr.accounts.GetAliasByID(utxo.AccountID)
		tmpUTXO.OutputID = fmt.Sprintf("%x", utxo.OutputID)
		tmpUTXO.AccountID = utxo.AccountID
		tmpUTXO.AssetID = fmt.Sprintf("%x", utxo.AssetID)
		tmpUTXO.Amount = utxo.Amount
		tmpUTXO.Change = utxo.Change
		tmpUTXO.Program = fmt.Sprintf("%x", utxo.Program)
		tmpUTXO.ProgramIndex = utxo.ProgramIndex
		tmpUTXO.RefData = fmt.Sprintf("%x", utxo.RefData)
		tmpUTXO.SourceID = fmt.Sprintf("%x", utxo.SourceID)
		tmpUTXO.SourcePos = utxo.SourcePos

		UTXOs = append(UTXOs, tmpUTXO)
	}

	return resWrapper(UTXOs)
}
