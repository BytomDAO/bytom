package api

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/net/http/httperror"
)

func init() {
	//Error code 050 represents alias of key duplicated
	errorFormatter.Errors[pseudohsm.ErrDuplicateKeyAlias] = httperror.Info{400, "BTM050", "Alias already exists"}
	//Error code 801 represents query request format error
	errorFormatter.Errors[pseudohsm.ErrInvalidAfter] = httperror.Info{400, "BTM801", "Invalid `after` in query"}
	//Error code 802 represents query reponses too many
	errorFormatter.Errors[pseudohsm.ErrTooManyAliasesToList] = httperror.Info{400, "BTM802", "Too many aliases to list"}
}

func (a *API) pseudohsmCreateKey(ctx context.Context, in struct {
	Alias    string `json:"alias"`
	Password string `json:"password"`
}) Response {
	xpub, err := a.wallet.Hsm.XCreate(in.Alias, in.Password)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(xpub)
}

func (a *API) pseudohsmListKeys(ctx context.Context) Response {
	return NewSuccessResponse(a.wallet.Hsm.ListKeys())
}

func (a *API) pseudohsmDeleteKey(ctx context.Context, x struct {
	Password string       `json:"password"`
	XPub     chainkd.XPub `json:"xpub"`
}) Response {
	if err := a.wallet.Hsm.XDelete(x.XPub, x.Password); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(nil)
}

type signResp struct {
	Tx           *txbuilder.Template `json:"transaction"`
	SignComplete bool                `json:"sign_complete"`
}

func (a *API) pseudohsmSignTemplates(ctx context.Context, x struct {
	Password string             `json:"password"`
	Txs      txbuilder.Template `json:"transaction"`
}) Response {
	if err := txbuilder.Sign(ctx, &x.Txs, x.Password, a.pseudohsmSignTemplate); err != nil {
		log.WithField("build err", err).Error("fail on sign transaction.")
		return NewErrorResponse(err)
	}
	log.Info("Sign Transaction complete.")
	return NewSuccessResponse(&signResp{Tx: &x.Txs, SignComplete: txbuilder.SignProgress(&x.Txs)})
}

func (a *API) pseudohsmSignTemplate(ctx context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte, password string) ([]byte, error) {
	return a.wallet.Hsm.XSign(xpub, path, data[:], password)
}

// ResetPasswordResp is response for reset password password
type ResetPasswordResp struct {
	Changed bool `json:"changed"`
}

func (a *API) pseudohsmResetPassword(ctx context.Context, ins struct {
	XPub        chainkd.XPub `json:"xpub"`
	OldPassword string       `json:"old_password"`
	NewPassword string       `json:"new_password"`
}) Response {
	resp := &ResetPasswordResp{Changed: false}
	if err := a.wallet.Hsm.ResetPassword(ins.XPub, ins.OldPassword, ins.NewPassword); err != nil {
		return NewSuccessResponse(resp)
	}
	resp.Changed = true
	return NewSuccessResponse(resp)
}
