package api

import (
	"context"

	"github.com/bytom/bytom/contract"
	"github.com/bytom/bytom/crypto/sha3pool"
	chainjson "github.com/bytom/bytom/encoding/json"
	"github.com/bytom/bytom/protocol/vm/vmutil"
)

// POST /create-asset
func (a *API) createContract(_ context.Context, ins struct {
	Alias    string             `json:"alias"`
	Contract chainjson.HexBytes `json:"contract"`
}) Response {
	var hash [32]byte
	sha3pool.Sum256(hash[:], ins.Contract)

	registerProgram, err := vmutil.RegisterProgram(ins.Contract)
	if err != nil {
		return NewErrorResponse(err)
	}

	callProgram, err := vmutil.CallContractProgram(hash[:])
	if err != nil {
		return NewErrorResponse(err)
	}

	c := &contract.Contract{
		Hash:            hash[:],
		Alias:           ins.Alias,
		Contract:        ins.Contract,
		CallProgram:     callProgram,
		RegisterProgram: registerProgram,
	}
	if err := a.wallet.ContractReg.SaveContract(c); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(c)
}

// POST /update-contract-alias
func (a *API) updateContractAlias(_ context.Context, ins struct {
	ID    chainjson.HexBytes `json:"id"`
	Alias string             `json:"alias"`
}) Response {
	if err := a.wallet.ContractReg.UpdateContract(ins.ID, ins.Alias); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(nil)
}

// POST /get-contract
func (a *API) getContract(_ context.Context, ins struct {
	ID chainjson.HexBytes `json:"id"`
}) Response {
	c, err := a.wallet.ContractReg.GetContract(ins.ID)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(c)
}

// POST /list-contracts
func (a *API) listContracts(_ context.Context) Response {
	cs, err := a.wallet.ContractReg.ListContracts()
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(cs)
}
