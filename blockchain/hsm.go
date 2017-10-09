package blockchain

import (
	"context"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/net/http/httperror"
	"github.com/bytom/net/http/httpjson"
)

func init() {
	errorFormatter.Errors[pseudohsm.ErrDuplicateKeyAlias] = httperror.Info{400, "BTM050", "Alias already exists"}
	errorFormatter.Errors[pseudohsm.ErrInvalidAfter] = httperror.Info{400, "BTM801", "Invalid `after` in query"}
	errorFormatter.Errors[pseudohsm.ErrTooManyAliasesToList] = httperror.Info{400, "BTM802", "Too many aliases to list"}
}

/*
// PseudoHSM configures the Core to expose the PseudoHSM endpoints. It
// is only included in non-production builds.
func PseudoHSM(hsm *Pseudohsm.HSM) RunOption {
	return func(api *API) {

		h := &pseudoHSMHandler{PseudoHSM: hsm}
		needConfig := api.needConfig()
		api.mux.Handle("/hsm/create-key", needConfig(h.pseudohsmCreateKey))
		api.mux.Handle("/hsm/list-keys", needConfig(h.pseudohsmListKeys))
		api.mux.Handle("/hsm/delete-key", needConfig(h.pseudohsmDeleteKey))
		api.mux.Handle("/hsm/sign-transaction", needConfig(h.pseudohsmSignTemplates))
		api.mux.Handle("/hsm/reset-password", needConfig(h.pseudohsmResetPassword))
		api.mux.Handle("/hsm/update-alias", needConfig(h.pseudohsmUpdateAlias))
	}
}


type pseudoHSMHandler struct {
	PseudoHSM *Pseudohsm.HSM
}
*/

func (a *BlockchainReactor) pseudohsmCreateKey(ctx context.Context, in struct{ Alias, Password string }) (result *pseudohsm.XPub, err error) {
	return a.hsm.XCreate(in.Password, in.Alias)
}

func (a *BlockchainReactor) pseudohsmListKeys(ctx context.Context, query requestQuery) (page, error) {
	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize // defGenericPageSize = 100
	}

	xpubs, after, err := a.hsm.ListKeys(query.After, limit)
	if err != nil {
		return page{}, err
	}

	var items []interface{}
	for _, xpub := range xpubs {
		items = append(items, xpub)
	}

	query.After = after

	return page{
		Items:    httpjson.Array(items),
		LastPage: len(xpubs) < limit,
		Next:     query,
	}, nil
}

func (a *BlockchainReactor) pseudohsmDeleteKey(ctx context.Context, x struct {
	Password string
	XPub     chainkd.XPub `json:"xpubs"`
}) error {
	return a.hsm.XDelete(x.XPub, x.Password)
}

func (a *BlockchainReactor) pseudohsmSignTemplates(ctx context.Context, x struct {
	Auth  string
	Txs   []*txbuilder.Template `json:"transactions"`
	XPubs []chainkd.XPub        `json:"xpubs"`
}) []interface{} {
	resp := make([]interface{}, 0, len(x.Txs))
	for _, tx := range x.Txs {
		err := txbuilder.Sign(ctx, tx, x.XPubs, x.Auth, a.pseudohsmSignTemplate)
		if err != nil {
			info := errorFormatter.Format(err)
			resp = append(resp, info)
		} else {
			resp = append(resp, tx)
		}
	}
	return resp
}

func (a *BlockchainReactor) pseudohsmSignTemplate(ctx context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte, password string) ([]byte, error) {
	sigBytes, err := a.hsm.XSign(xpub, path, data[:], password)
	if err == pseudohsm.ErrNoKey {
		return nil, nil
	}
	return sigBytes, err
}

func (a *BlockchainReactor) pseudohsmResetPassword(ctx context.Context, x struct {
	OldPassword string
	NewPassword string
	XPub        chainkd.XPub `json:"xpubs"`
}) error {
	return a.hsm.ResetPassword(x.XPub, x.OldPassword, x.NewPassword)
}

func (a *BlockchainReactor) pseudohsmUpdateAlias(ctx context.Context, x struct {
	Password string
	NewAlias string
	XPub     chainkd.XPub `json:"xpubs"`
}) error {
	return a.hsm.UpdateAlias(x.XPub, x.Password, x.NewAlias)
}
