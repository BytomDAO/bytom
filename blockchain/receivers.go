package blockchain

import (
	"context"
)

func (bcr *BlockchainReactor) createAccountAddress(ctx context.Context, ins struct {
	AccountInfo string `json:"account_info"`
}) Response {
	receiver, err := bcr.accounts.CreateAddressReceiver(ctx, ins.AccountInfo)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(receiver)
}

func (bcr *BlockchainReactor) createAccountPubkey(ctx context.Context, ins struct {
	AccountInfo string `json:"account_info"`
}) Response {
	pubkeyInfo, err := bcr.accounts.CreatePubkeyInfo(nil, ins.AccountInfo)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(pubkeyInfo)
}
