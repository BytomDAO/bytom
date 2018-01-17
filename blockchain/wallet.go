package blockchain

import (
	"context"

	"github.com/bytom/crypto/ed25519/chainkd"
)

func (bcr *BlockchainReactor) walletExportKey(ctx context.Context, in struct {
	Password string       `json:"password"`
	XPub     chainkd.XPub `json:"xpub"`
}) Response {
	key, err := bcr.wallet.ExportAccountPrivKey(bcr.hsm, in.XPub, in.Password)
	if err != nil {
		return NewErrorResponse(err)
	}

	type privateKey struct {
		PrivateKey string `json:"private_key"`
	}
	return NewSuccessResponse(privateKey{PrivateKey: *key})
}

func (bcr *BlockchainReactor) walletImportKey(ctx context.Context, in struct {
	KeyAlias     string       `json:"key_alias"`
	Password     string       `json:"password"`
	XPrv         chainkd.XPrv `json:"xprv"`
	Index        uint64       `json:"index"`
	AccountAlias string       `json:"account_alias"`
}) Response {
	xpub, err := bcr.wallet.ImportAccountPrivKey(bcr.hsm, in.XPrv, in.KeyAlias, in.Password, in.Index, in.AccountAlias)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(xpub)
}
