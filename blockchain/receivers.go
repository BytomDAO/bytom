package blockchain

import (
	"context"
)

func (bcr *BlockchainReactor) createAccountReceiver(ctx context.Context, ins struct {
	AccountInfo string `json:"account_info"`
}) Response {
	receiver, err := bcr.accounts.CreateAccountReceiver(ctx, ins.AccountInfo)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(receiver)
}
