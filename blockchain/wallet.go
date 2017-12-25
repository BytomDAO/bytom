package blockchain

import (
	"context"

	"github.com/bytom/crypto/ed25519/chainkd"
)

func (a *BlockchainReactor) walletExportKey(ctx context.Context, in struct {
	Password string
	XPub     chainkd.XPub
}) interface{} {
	key, err := a.wallet.ExportAccountPrivKey(a.hsm, in.XPub, in.Password)
	if err != nil {
		return err.Error()
	}
	return key
}

func (a *BlockchainReactor) walletImportKey(ctx context.Context, in struct {
	Alias    string
	Password string
	XPrv     chainkd.XPrv
	Index    uint64
}) interface{} {
	xpub, err := a.wallet.ImportAccountPrivKey(a.hsm, in.XPrv, in.Alias, in.Password, in.Index)
	if err != nil {
		return err.Error()
	}
	return xpub
}
