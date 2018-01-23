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

func (bcr *BlockchainReactor) createAccountContract(ctx context.Context, ins struct {
	AccountInfo     string `json:"account_info"`
	ContractProgram string `json:"contract_program"`
}) Response {
	contract, err := bcr.accounts.CreateContractInfo(nil, ins.AccountInfo, ins.ContractProgram)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(contract)
}
