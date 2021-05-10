package api

import (
	"context"
	"fmt"

	"github.com/bytom/bytom/blockchain/query"
)

// POST /list-address-balances
func (a *API) listAddressBalances(ctx context.Context, req struct {
	Address string `json:"address"`
}) Response {
	balances, err := a.wallet.GetAddressBalances(req.Address)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(balances)
}

// POST /list-address-unspent-outputs
func (a *API) listAddressUnspentOutputs(ctx context.Context, req struct {
	Address       string `json:"address"`
	ID            string `json:"id"`
	Unconfirmed   bool   `json:"unconfirmed"`
	SmartContract bool   `json:"smart_contract"`
	From          uint   `json:"from"`
	Count         uint   `json:"count"`
}) Response {
	addressUTXOs := a.wallet.GetAddressUTXOs(req.Address, req.ID, req.Unconfirmed, req.SmartContract)
	utxos := []query.AnnotatedUTXO{}
	for _, utxo := range addressUTXOs {
		utxos = append([]query.AnnotatedUTXO{{
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
		}}, utxos...)
	}
	start, end := getPageRange(len(utxos), req.From, req.Count)
	return NewSuccessResponse(utxos[start:end])
}
