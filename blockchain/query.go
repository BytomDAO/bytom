package blockchain

import (
	"context"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/query"
)

// POST /list-accounts
func (bcr *BlockchainReactor) listAccounts(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	accounts, err := bcr.accounts.ListAccounts(filter.ID)
	if err != nil {
		log.Errorf("listAccounts: %v", err)
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(accounts)
}

// POST /list-assets
func (bcr *BlockchainReactor) listAssets(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	assets, err := bcr.assets.ListAssets(filter.ID)
	if err != nil {
		log.Errorf("listAssets: %v", err)
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(assets)
}

// POST /listBalances
func (bcr *BlockchainReactor) listBalances(ctx context.Context) Response {
	accountUTXOs, err := bcr.wallet.GetAccountUTXOs("", false)
	if err != nil {
		log.Errorf("GetAccountUTXOs: %v", err)
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(bcr.indexBalances(accountUTXOs))
}

type accountBalance struct {
	AccountID  string `json:"account_id"`
	Alias      string `json:"account_alias"`
	AssetAlias string `json:"asset_alias"`
	AssetID    string `json:"asset_id"`
	Amount     uint64 `json:"amount"`
}

func (bcr *BlockchainReactor) indexBalances(accountUTXOs []account.UTXO) []accountBalance {
	accBalance := make(map[string]map[string]uint64)
	balances := make([]accountBalance, 0)
	tmpBalance := accountBalance{}

	for _, accountUTXO := range accountUTXOs {

		assetID := accountUTXO.AssetID.String()
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

		for _, assetID := range sortedAsset {

			alias := bcr.accounts.GetAliasByID(id)
			assetAlias := bcr.assets.GetAliasByID(assetID)
			tmpBalance.Alias = alias
			tmpBalance.AccountID = id
			tmpBalance.AssetID = assetID
			tmpBalance.AssetAlias = assetAlias
			tmpBalance.Amount = accBalance[id][assetID]
			balances = append(balances, tmpBalance)
		}
	}

	return balances
}

// POST /list-transactions
func (bcr *BlockchainReactor) listTransactions(ctx context.Context, filter struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Detail    bool   `json:"detail"`
}) Response {
	var transactions []*query.AnnotatedTx
	var err error

	if filter.AccountID != "" {
		transactions, err = bcr.wallet.GetTransactionsByAccountID(filter.AccountID)
	} else {
		transactions, err = bcr.wallet.GetTransactionsByTxID(filter.ID)
	}

	if err != nil {
		log.Errorf("listTransactions: %v", err)
		return NewErrorResponse(err)
	}

	if filter.Detail == false {
		txSummary := bcr.wallet.GetTransactionsSummary(transactions)
		return NewSuccessResponse(txSummary)
	}
	return NewSuccessResponse(transactions)
}

type annotatedUTXO struct {
	Alias               string `json:"account_alias"`
	OutputID            string `json:"id"`
	AssetID             string `json:"asset_id"`
	AssetAlias          string `json:"asset_alias"`
	Amount              uint64 `json:"amount"`
	AccountID           string `json:"account_id"`
	Address             string `json:"address"`
	ControlProgramIndex uint64 `json:"control_program_index"`
	Program             string `json:"program"`
	SourceID            string `json:"source_id"`
	SourcePos           uint64 `json:"source_pos"`
	RefDataHash         string `json:"ref_data"`
	ValidHeight         uint64 `json:"valid_height"`
}

// POST /list-unspent-outputs
func (bcr *BlockchainReactor) listUnspentOutputs(ctx context.Context, filter struct {
	ID            string `json:"id"`
	SmartContract bool   `json:"smart_contract"`
}) Response {
	tmpUTXO := annotatedUTXO{}
	UTXOs := make([]annotatedUTXO, 0)

	accountUTXOs, err := bcr.wallet.GetAccountUTXOs(filter.ID, filter.SmartContract)
	if err != nil {
		log.Errorf("list Unspent Outputs: %v", err)
		return NewErrorResponse(err)
	}

	for _, utxo := range accountUTXOs {
		tmpUTXO.AccountID = utxo.AccountID
		tmpUTXO.OutputID = utxo.OutputID.String()
		tmpUTXO.SourceID = utxo.SourceID.String()
		tmpUTXO.AssetID = utxo.AssetID.String()
		tmpUTXO.Amount = utxo.Amount
		tmpUTXO.SourcePos = utxo.SourcePos
		tmpUTXO.Program = fmt.Sprintf("%x", utxo.ControlProgram)
		tmpUTXO.RefDataHash = utxo.RefDataHash.String()
		tmpUTXO.ControlProgramIndex = utxo.ControlProgramIndex
		tmpUTXO.Address = utxo.Address
		tmpUTXO.ValidHeight = utxo.ValidHeight

		tmpUTXO.Alias = bcr.accounts.GetAliasByID(utxo.AccountID)
		tmpUTXO.AssetAlias = bcr.assets.GetAliasByID(tmpUTXO.AssetID)

		UTXOs = append(UTXOs, tmpUTXO)
	}

	return NewSuccessResponse(UTXOs)
}
