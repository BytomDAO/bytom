package blockchain

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/query"
)

// POST /list-accounts
func (bcr *BlockchainReactor) listAccounts(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	accounts, err := bcr.wallet.AccountMgr.ListAccounts(filter.ID)
	if err != nil {
		log.Errorf("listAccounts: %v", err)
		return NewErrorResponse(err)
	}

	annotatedAccounts := make([]query.AnnotatedAccount, 0, len(accounts))
	for _, acc := range accounts {
		annotated, err := account.Annotated(acc)
		if err != nil {
			return NewErrorResponse(err)
		}

		annotatedAccounts = append(annotatedAccounts, *annotated)
	}

	return NewSuccessResponse(annotatedAccounts)
}

// POST /list-assets
func (bcr *BlockchainReactor) listAssets(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	assets, err := bcr.wallet.AssetReg.ListAssets(filter.ID)
	if err != nil {
		log.Errorf("listAssets: %v", err)
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(assets)
}

// POST /listBalances
func (bcr *BlockchainReactor) listBalances(ctx context.Context) Response {
	if balances, err := bcr.wallet.GetAccountBalances(""); err != nil {
		log.Errorf("GetAccountUTXOs: %v", err)
		return NewErrorResponse(err)
	} else {
		return NewSuccessResponse(balances)
	}
}

// POST /get-transaction
func (bcr *BlockchainReactor) getTransaction(ctx context.Context, txInfo struct {
	TxID string `json:"tx_id"`
}) Response {
	transaction, err := bcr.wallet.GetTransactionByTxID(txInfo.TxID)
	if err != nil {
		log.Errorf("getTransaction error: %v", err)
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(transaction)
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

// POST /list-unspent-outputs
func (bcr *BlockchainReactor) listUnspentOutputs(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	accountUTXOs, err := bcr.wallet.GetAccountUTXOs(filter.ID)
	if err != nil {
		log.Errorf("list Unspent Outputs: %v", err)
		return NewErrorResponse(err)
	}

	var UTXOs []query.AnnotatedUTXO
	for _, utxo := range accountUTXOs {
		UTXOs = append(UTXOs, query.AnnotatedUTXO{
			AccountID:           utxo.AccountID,
			OutputID:            utxo.OutputID.String(),
			SourceID:            utxo.SourceID.String(),
			AssetID:             utxo.AssetID.String(),
			Amount:              utxo.Amount,
			SourcePos:           utxo.SourcePos,
			Program:             fmt.Sprintf("%x", utxo.ControlProgram),
			ControlProgramIndex: utxo.ControlProgramIndex,
			Address:             utxo.Address,
			ValidHeight:         utxo.ValidHeight,
			Alias:               bcr.wallet.AccountMgr.GetAliasByID(utxo.AccountID),
			AssetAlias:          bcr.wallet.AssetReg.GetAliasByID(utxo.AssetID.String()),
		})
	}

	return NewSuccessResponse(UTXOs)
}
