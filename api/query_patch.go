package api

import (
	"context"
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
