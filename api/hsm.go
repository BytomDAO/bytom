package api

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/blockchain/txbuilder"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
)

type createKeyResp struct {
	Alias    string       `json:"alias"`
	XPub     chainkd.XPub `json:"xpub"`
	File     string       `json:"file"`
	Mnemonic string       `json:"mnemonic,omitempty"`
}

func (a *API) pseudohsmCreateKey(ctx context.Context, in struct {
	Alias    string `json:"alias"`
	Password string `json:"password"`
	Mnemonic string `json:"mnemonic"`
	Language string `json:"language"`
}) Response {
	if in.Language == "" {
		in.Language = "en"
	}
	if len(in.Mnemonic) > 0 {
		xpub, err := a.wallet.Hsm.ImportKeyFromMnemonic(in.Alias, in.Password, in.Mnemonic, in.Language)
		if err != nil {
			return NewErrorResponse(err)
		}
		return NewSuccessResponse(&createKeyResp{Alias: xpub.Alias, XPub: xpub.XPub, File: xpub.File})
	}
	xpub, mnemonic, err := a.wallet.Hsm.XCreate(in.Alias, in.Password, in.Language)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(&createKeyResp{Alias: xpub.Alias, XPub: xpub.XPub, File: xpub.File, Mnemonic: *mnemonic})
}

func (a *API) pseudohsmUpdateKeyAlias(ctx context.Context, in struct {
	XPub     chainkd.XPub `json:"xpub"`
	NewAlias string       `json:"new_alias"`
}) Response {
	if err := a.wallet.Hsm.UpdateKeyAlias(in.XPub, in.NewAlias); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(nil)
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

type signTemplateResp struct {
	Tx           *txbuilder.Template `json:"transaction"`
	SignComplete bool                `json:"sign_complete"`
}

func (a *API) signTemplate(ctx context.Context, x struct {
	Password string             `json:"password"`
	Txs      txbuilder.Template `json:"transaction"`
}) Response {
	if err := txbuilder.Sign(ctx, &x.Txs, x.Password, a.pseudohsmSignTemplate); err != nil {
		log.WithField("build err", err).Error("fail on sign transaction.")
		return NewErrorResponse(err)
	}
	log.Info("Sign Transaction complete.")
	return NewSuccessResponse(&signTemplateResp{Tx: &x.Txs, SignComplete: txbuilder.SignProgress(&x.Txs)})
}

type signTemplatesResp struct {
	Tx           []*txbuilder.Template `json:"transaction"`
	SignComplete bool                  `json:"sign_complete"`
}

func (a *API) signTemplates(ctx context.Context, x struct {
	Password string                `json:"password"`
	Txs      []*txbuilder.Template `json:"transactions"`
}) Response {
	signComplete := true
	for _, tx := range x.Txs {
		if err := txbuilder.Sign(ctx, tx, x.Password, a.pseudohsmSignTemplate); err != nil {
			log.WithField("build err", err).Error("fail on sign transaction.")
			return NewErrorResponse(err)
		}
		signComplete = signComplete && txbuilder.SignProgress(tx)
	}

	log.Info("Sign Chain Tx complete.")
	return NewSuccessResponse(&signTemplatesResp{Tx: x.Txs, SignComplete: signComplete})
}

func (a *API) pseudohsmSignTemplate(ctx context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte, password string) ([]byte, error) {
	return a.wallet.Hsm.XSign(xpub, path, data[:], password)
}

// ResetPasswordResp is response for reset key password
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

// CheckPasswordResp is response for check key password
type CheckPasswordResp struct {
	CheckResult bool `json:"check_result"`
}

func (a *API) pseudohsmCheckPassword(ctx context.Context, ins struct {
	XPub     chainkd.XPub `json:"xpub"`
	Password string       `json:"password"`
}) Response {
	resp := &CheckPasswordResp{CheckResult: false}
	if _, err := a.wallet.Hsm.LoadChainKDKey(ins.XPub, ins.Password); err != nil {
		return NewSuccessResponse(resp)
	}
	resp.CheckResult = true
	return NewSuccessResponse(resp)
}
