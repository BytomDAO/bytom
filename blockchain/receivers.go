package blockchain

import (
	"context"
	"time"
)

// POST /create-account-receiver
func (a *BlockchainReactor) createAccountReceiver(ctx context.Context, ins struct {
	AccountInfo string    `json:"account_info"`
	ExpiresAt   time.Time `json:"expires_at"`
}) interface{} {
	var response interface{}

	receiver, err := a.accounts.CreateReceiver(nil, ins.AccountInfo, ins.ExpiresAt)
	if err != nil {
		response = err
	} else {
		response = receiver
	}

	return response
}

func (a *BlockchainReactor) createAccountAddress(ctx context.Context, ins struct {
	AccountInfo string    `json:"account_info"`
	ExpiresAt   time.Time `json:"expires_at"`
}) interface{} {
	receiver, err := a.accounts.CreateAddress(ctx, ins.AccountInfo, ins.ExpiresAt)
	if err != nil {
		return err
	}
	return receiver
}
