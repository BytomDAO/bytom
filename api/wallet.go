package api

import (
	"github.com/bytom/errors"
)

// POST /wallet error
func (a *API) walletError() Response {
	return NewErrorResponse(errors.New("wallet not found, please check that the wallet is open"))
}
