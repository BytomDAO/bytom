package blockchain

import (
	"context"

	"github.com/bytom/crypto/ed25519/chainkd"
)

func (bcr *BlockchainReactor) walletExportKey(ctx context.Context, in struct {
	Password string
	XPub     chainkd.XPub
}) interface{} {
	key, err := bcr.wallet.ExportAccountPrivKey(bcr.hsm, in.XPub, in.Password)
	if err != nil {
		return err.Error()
	}
	return key
}

func (bcr *BlockchainReactor) walletImportKey(ctx context.Context, in struct {
	Alias    string
	Password string
	XPrv     chainkd.XPrv
	Index    uint64
}) interface{} {
	xpub, err := bcr.wallet.ImportAccountPrivKey(bcr.hsm, in.XPrv, in.Alias, in.Password, in.Index)
	if err != nil {
		return err.Error()
	}
	return xpub
}
