package blockchain

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

func (a *BlockchainReactor) pseudohsmCreateKey(ctx context.Context, in struct {
	Alias    string `json:"alias"`
	Password string `json:"password"`
}) Response {
	xpub, err := a.hsm.XCreate(in.Alias, in.Password)
	if err != nil {
		return resWrapper(nil, err)
	}
	return resWrapper(xpub)
}

func (a *BlockchainReactor) pseudohsmListKeys(ctx context.Context) Response {
	xpubs, err := a.hsm.ListKeys()
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(xpubs)
}

func (a *BlockchainReactor) pseudohsmDeleteKey(ctx context.Context, x struct {
	Password string       `json:"password"`
	XPub     chainkd.XPub `json:"xpubs"`
}) Response {
	if err := a.hsm.XDelete(x.XPub, x.Password); err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(nil)
}

func (a *BlockchainReactor) pseudohsmSignTemplates(ctx context.Context, x struct {
	Auth string             `json:"auth"`
	Txs  txbuilder.Template `json:"transaction"`
}) Response {
	var err error
	if err = txbuilder.Sign(ctx, &x.Txs, nil, x.Auth, a.pseudohsmSignTemplate); err != nil {
		log.WithField("build err", err).Error("fail on sign transaction.")
		return resWrapper(nil, err)
	}

	log.Info("Sign Transaction complete.")
	return resWrapper(&x.Txs)
}

func (a *BlockchainReactor) pseudohsmSignTemplate(ctx context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte, password string) ([]byte, error) {
	sigBytes, err := a.hsm.XSign(xpub, path, data[:], password)
	if err == pseudohsm.ErrNoKey {
		return nil, err
	}
	return sigBytes, nil
}

func (a *BlockchainReactor) pseudohsmResetPassword(ctx context.Context, x struct {
	OldPassword string
	NewPassword string
	XPub        chainkd.XPub `json:"xpubs"`
}) error {
	return a.hsm.ResetPassword(x.XPub, x.OldPassword, x.NewPassword)
}
