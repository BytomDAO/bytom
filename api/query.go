package api

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"encoding/hex"
	"github.com/bytom/account"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
)

// POST /list-accounts
func (a *API) listAccounts(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	accounts, err := a.wallet.AccountMgr.ListAccounts(filter.ID)
	if err != nil {
		log.Errorf("listAccounts: %v", err)
		return NewErrorResponse(err)
	}

	var annotatedAccounts []query.AnnotatedAccount
	for _, acc := range accounts {
		annotatedAccounts = append(annotatedAccounts, *account.Annotated(acc))
	}

	return NewSuccessResponse(annotatedAccounts)
}

// POST /list-assets
func (a *API) listAssets(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	assets, err := a.wallet.AssetReg.ListAssets(filter.ID)
	if err != nil {
		log.Errorf("listAssets: %v", err)
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(assets)
}

// POST /listBalances
func (a *API) listBalances(ctx context.Context) Response {
	return NewSuccessResponse(a.wallet.GetAccountBalances(""))
}

// POST /get-transaction
func (a *API) getTransaction(ctx context.Context, txInfo struct {
	TxID string `json:"tx_id"`
}) Response {
	transaction, err := a.wallet.GetTransactionByTxID(txInfo.TxID)
	if err != nil {
		log.Errorf("getTransaction error: %v", err)
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(transaction)
}

// POST /list-transactions
func (a *API) listTransactions(ctx context.Context, filter struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Detail    bool   `json:"detail"`
}) Response {
	var transactions []*query.AnnotatedTx
	var err error

	if filter.AccountID != "" {
		transactions, err = a.wallet.GetTransactionsByAccountID(filter.AccountID)
	} else {
		transactions, err = a.wallet.GetTransactionsByTxID(filter.ID)
	}

	if err != nil {
		log.Errorf("listTransactions: %v", err)
		return NewErrorResponse(err)
	}

	if filter.Detail == false {
		txSummary := a.wallet.GetTransactionsSummary(transactions)
		return NewSuccessResponse(txSummary)
	}
	return NewSuccessResponse(transactions)
}

// POST /get-unconfirmed-transaction
func (a *API) getUnconfirmedTx(ctx context.Context, filter struct {
	TxID string `json:"tx_id"`
}) Response {
	txID, err := hex.DecodeString(filter.TxID)
	if err != nil {
		log.Errorf("convert txID[%s] string to byte err: %v", filter.TxID, err)
		return NewErrorResponse(err)
	}

	var tmpTxID [32]byte
	copy(tmpTxID[:], txID[:])

	txHash := bc.NewHash(tmpTxID)
	txPool := a.chain.GetTxPool()
	txDesc, err := txPool.GetTransaction(&txHash)
	if err != nil {
		return NewErrorResponse(err)
	}

	tx := &BlockTx{
		ID:         txDesc.Tx.ID,
		Version:    txDesc.Tx.Version,
		Size:       txDesc.Tx.SerializedSize,
		TimeRange:  txDesc.Tx.TimeRange,
		Inputs:     []*query.AnnotatedInput{},
		Outputs:    []*query.AnnotatedOutput{},
		StatusFail: false,
	}

	for i := range txDesc.Tx.Inputs {
		tx.Inputs = append(tx.Inputs, a.wallet.BuildAnnotatedInput(txDesc.Tx, uint32(i)))
	}
	for i := range txDesc.Tx.Outputs {
		tx.Outputs = append(tx.Outputs, a.wallet.BuildAnnotatedOutput(txDesc.Tx, i))
	}

	return NewSuccessResponse(tx)
}

type getTxPoolResp struct {
	TxID bc.Hash `json:"tx_id"`
}

// POST /list-unconform-transactions
func (a *API) listUnconformTxs(ctx context.Context) Response {
	txIDs := []getTxPoolResp{}

	txPool := a.chain.GetTxPool()
	txs := txPool.GetTransactions()
	for _, txDesc := range txs {
		txIDs = append(txIDs, getTxPoolResp{TxID: bc.Hash(txDesc.Tx.ID)})
	}

	return NewSuccessResponse(txIDs)
}

// POST /list-unspent-outputs
func (a *API) listUnspentOutputs(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	accountUTXOs := a.wallet.GetAccountUTXOs(filter.ID)

	var UTXOs []query.AnnotatedUTXO
	for _, utxo := range accountUTXOs {
		UTXOs = append([]query.AnnotatedUTXO{{
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
			Alias:               a.wallet.AccountMgr.GetAliasByID(utxo.AccountID),
			AssetAlias:          a.wallet.AssetReg.GetAliasByID(utxo.AssetID.String()),
			Change:              utxo.Change,
		}}, UTXOs...)
	}

	return NewSuccessResponse(UTXOs)
}

// return gasRate
func (a *API) gasRate() Response {
	gasrate := map[string]int64{"gasRate": consensus.VMGasRate}
	return NewSuccessResponse(gasrate)
}
