package api

import (
	"errors"
)

func (a *API) setMining(in struct {
	IsMining bool `json:"is_mining"`
}) Response {
	if in.IsMining {
		if _, err := a.wallet.AccountMgr.GetMiningAddress(); err != nil {
			return NewErrorResponse(errors.New("Mining address does not exist"))
		}
		return a.startMining()
	}
	return a.stopMining()
}

func (a *API) startMining() Response {
	a.blockProposer.Start()
	if !a.IsMining() {
		return NewErrorResponse(errors.New("Failed to start mining"))
	}
	return NewSuccessResponse("")
}

func (a *API) stopMining() Response {
	a.blockProposer.Stop()
	if a.IsMining() {
		return NewErrorResponse(errors.New("Failed to stop mining"))
	}
	return NewSuccessResponse("")
}
