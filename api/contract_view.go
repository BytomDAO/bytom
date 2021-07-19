package api

import (
	"context"
)

type ContractId struct {
	uuid string `json:"uuid"`
}

type Utxo struct {
	txHash    string `json:"tx_hash"`
	Asset     string `json:"asset"`
	Amount    uint64 `json:"amount"`
	Program   string `json:"program"`
	StateData string `json:"stateData"`
	// status     string `json:"status"` // confirmed,unconfirmed,emerged
}

type GetContractUtxosResp struct {
	cus []*ConfirmedContractUtxos
	uus []*UnconfirmedContractUtxos
	eus []*EmergedContractUtxos
}

// status is confirmed
type ConfirmedContractUtxos struct {
	cus []*Utxo
}

// status is unconfirmed
type UnconfirmedContractUtxos struct {
	uus []*Utxo
}

// status is emerged
type EmergedContractUtxos struct {
	eus []*Utxo
}

func (a *API) createContractView(_ context.Context, ins struct {
	blockHash string `json:"block_hash"`
	txHash    string `json:"tx_hash"`
}) Response {
	return NewSuccessResponse(nil)
}

func (a *API) deleteContractView(_ context.Context, uuid *ContractId) Response {
	return NewSuccessResponse(nil)
}

func (a *API) getContractUtxos(_ context.Context, uuid *ContractId) Response {
	return NewSuccessResponse(nil)
}

func (a *API) listContractUtxos(_ context.Context, uuid *ContractId) Response {
	return NewSuccessResponse(nil)
}

func (a *API) buildContractTx(_ context.Context, uuid *ContractId) Response {
	return NewSuccessResponse(nil)
}
