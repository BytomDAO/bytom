package blockchain

import (
	"bytes"
	"context"

	"github.com/tendermint/go-wire/data/base58"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
)

//KeyImportParams private key import param
type KeyImportParams struct {
	KeyAlias     string `json:"alias"`
	Password     string `json:"password"`
	XPrv         string `json:"xprv"`
	Index        uint64 `json:"index"`
	AccountAlias string `json:"account_alias"`
}

func (a *API) walletExportKey(ctx context.Context, in struct {
	Password string       `json:"password"`
	XPub     chainkd.XPub `json:"xpub"`
}) Response {
	key, err := a.wallet.ExportAccountPrivKey(in.XPub, in.Password)
	if err != nil {
		return NewErrorResponse(err)
	}

	type privateKey struct {
		PrivateKey string `json:"private_key"`
	}
	return NewSuccessResponse(privateKey{PrivateKey: *key})
}

func (a *API) walletImportKey(ctx context.Context, in KeyImportParams) Response {
	rawData, err := base58.Decode(in.XPrv)
	if err != nil {
		return NewErrorResponse(err)
	}

	if len(rawData) != 68 {
		return NewErrorResponse(errors.New("invalid private key hash length"))
	}

	var hashed [32]byte
	sha3pool.Sum256(hashed[:], rawData[:64])
	if res := bytes.Compare(hashed[:4], rawData[64:]); res != 0 {
		return NewErrorResponse(errors.New("private hash error"))
	}

	var xprv [64]byte
	copy(xprv[:], rawData[:64])

	xpub, err := a.wallet.ImportAccountPrivKey(xprv, in.KeyAlias, in.Password, in.Index, in.AccountAlias)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(xpub)
}

func (a *API) keyImportProgress(ctx context.Context) Response {
	data, err := a.wallet.GetRescanStatus()
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(data)
}
