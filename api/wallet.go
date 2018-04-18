package api

import (
	"context"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/errors"
)

// POST /wallet error
func (a *API) walletError() Response {
	return NewErrorResponse(errors.New("wallet not found, please check that the wallet is open"))
}

type WalletImage struct {
	AccountImage *account.AccountImage
	AssetImage   *asset.AssetImage
	KeyImages    []*pseudohsm.KeyImage
}

func (a *API) restoreWalletImage(ctx context.Context, image WalletImage) Response {
	if err := a.wallet.Hsm.Restore(image.KeyImages); err != nil {
		return NewErrorResponse(err)
	}
	if err := a.wallet.AssetReg.Restore(image.AssetImage); err != nil {
		return NewErrorResponse(err)
	}
	if err := a.wallet.AccountMgr.Restore(image.AccountImage); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(nil)
}

func (a *API) backupWalletImage() Response {
	keyImages, err := a.wallet.Hsm.Backup()
	if err != nil {
		return NewErrorResponse(err)
	}
	assetImage, err := a.wallet.AssetReg.Backup()
	if err != nil {
		return NewErrorResponse(err)
	}
	accountImage, err := a.wallet.AccountMgr.Backup()
	if err != nil {
		return NewErrorResponse(err)
	}

	image := &WalletImage{
		KeyImages:    keyImages,
		AssetImage:   assetImage,
		AccountImage: accountImage,
	}
	return NewSuccessResponse(image)
}
