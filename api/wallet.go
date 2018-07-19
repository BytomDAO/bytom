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

// WalletImage hold the ziped wallet data
type WalletImage struct {
	AccountImage *account.Image      `json:"account_image"`
	AssetImage   *asset.Image        `json:"asset_image"`
	KeyImages    *pseudohsm.KeyImage `json:"key_images"`
}

func (a *API) restoreWalletImage(ctx context.Context, image WalletImage) Response {
	if err := a.wallet.Hsm.Restore(image.KeyImages); err != nil {
		return NewErrorResponse(errors.Wrap(err, "restore key images"))
	}
	if err := a.wallet.AssetReg.Restore(image.AssetImage); err != nil {
		return NewErrorResponse(errors.Wrap(err, "restore asset image"))
	}
	if err := a.wallet.AccountMgr.Restore(image.AccountImage); err != nil {
		return NewErrorResponse(errors.Wrap(err, "restore account image"))
	}
	a.wallet.RescanBlocks()
	return NewSuccessResponse(nil)
}

func (a *API) backupWalletImage() Response {
	keyImages, err := a.wallet.Hsm.Backup()
	if err != nil {
		return NewErrorResponse(errors.Wrap(err, "backup key images"))
	}
	assetImage, err := a.wallet.AssetReg.Backup()
	if err != nil {
		return NewErrorResponse(errors.Wrap(err, "backup asset image"))
	}
	accountImage, err := a.wallet.AccountMgr.Backup()
	if err != nil {
		return NewErrorResponse(errors.Wrap(err, "backup account image"))
	}

	image := &WalletImage{
		KeyImages:    keyImages,
		AssetImage:   assetImage,
		AccountImage: accountImage,
	}
	return NewSuccessResponse(image)
}

func (a *API) rescanWallet() Response {
	a.wallet.RescanBlocks()
	return NewSuccessResponse(nil)
}

// WalletInfo return wallet information
type WalletInfo struct {
	BestBlockHeight uint64 `json:"best_block_height"`
	WalletHeight    uint64 `json:"wallet_height"`
}

func (a *API) getWalletInfo() Response {
	bestBlockHeight := a.chain.BestBlockHeight()
	walletStatus := a.wallet.GetWalletStatusInfo()

	return NewSuccessResponse(&WalletInfo{
		BestBlockHeight: bestBlockHeight,
		WalletHeight:    walletStatus.WorkHeight,
	})
}
