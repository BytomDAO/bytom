package blockchain

import (
	"context"
	"time"
)

// POST /create-account-receiver
func (bcr *BlockchainReactor) createAccountReceiver(ctx context.Context, ins struct {
	AccountInfo string    `json:"account_info"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}) Response {
	receiver, err := bcr.accounts.CreateReceiver(nil, ins.AccountInfo, ins.ExpiresAt)
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(*receiver)
}

func (bcr *BlockchainReactor) createAccountAddress(ctx context.Context, ins struct {
	AccountInfo string    `json:"account_info"`
}) Response {
	receiver, err := bcr.accounts.CreateAddressReceiver(ctx, ins.AccountInfo)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(receiver)
}
