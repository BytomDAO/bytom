package api

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/account"
	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol/vm/vmutil"
)

// POST /create-account
func (a *API) createAccount(ctx context.Context, ins struct {
	RootXPubs []chainkd.XPub         `json:"root_xpubs"`
	Quorum    int                    `json:"quorum"`
	Alias     string                 `json:"alias"`
}) Response {
	acc, err := a.wallet.AccountMgr.Create(ctx, ins.RootXPubs, ins.Quorum, ins.Alias)
	if err != nil {
		return NewErrorResponse(err)
	}

	annotatedAccount := account.Annotated(acc)
	log.WithField("account ID", annotatedAccount.ID).Info("Created account")

	return NewSuccessResponse(annotatedAccount)
}

// POST /delete-account
func (a *API) deleteAccount(ctx context.Context, in struct {
	AccountInfo string `json:"account_info"`
}) Response {
	if err := a.wallet.AccountMgr.DeleteAccount(in); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(nil)
}

type validateAddressResp struct {
	Vaild   bool `json:"vaild"`
	IsLocal bool `json:"is_local"`
}

// POST /validate-address
func (a *API) validateAddress(ctx context.Context, ins struct {
	Address string `json:"address"`
}) Response {
	resp := &validateAddressResp{
		Vaild:   false,
		IsLocal: false,
	}
	address, err := common.DecodeAddress(ins.Address, &consensus.MainNetParams)
	if err != nil {
		return NewSuccessResponse(resp)
	}

	redeemContract := address.ScriptAddress()
	program := []byte{}
	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return NewSuccessResponse(resp)
	}
	if err != nil {
		return NewSuccessResponse(resp)
	}

	resp.Vaild = true
	resp.IsLocal = a.wallet.AccountMgr.IsLocalControlProgram(program)
	return NewSuccessResponse(resp)
}

type addressResp struct {
	AccountAlias string `json:"account_alias"`
	AccountID    string `json:"account_id"`
	Address      string `json:"address"`
	Change       bool   `json:"change"`
}

func (a *API) listAddresses(ctx context.Context, ins struct {
	AccountID    string `json:"account_id"`
	AccountAlias string `json:"account_alias"`
}) Response {
	accountID := ins.AccountID
	if ins.AccountAlias != "" {
		acc, err := a.wallet.AccountMgr.FindByAlias(ctx, ins.AccountAlias)
		if err != nil {
			return NewErrorResponse(err)
		}

		accountID = acc.ID
	}

	cps, err := a.wallet.AccountMgr.ListControlProgram()
	if err != nil {
		return NewErrorResponse(err)
	}

	var addresses []*addressResp
	for _, cp := range cps {
		if cp.Address == "" || (len(accountID) != 0 && strings.Compare(accountID, cp.AccountID) != 0) {
			continue
		}

		accountAlias := a.wallet.AccountMgr.GetAliasByID(cp.AccountID)
		addresses = append(addresses, &addressResp{
			AccountAlias: accountAlias,
			AccountID:    cp.AccountID,
			Address:      cp.Address,
			Change:       cp.Change,
		})
	}
	return NewSuccessResponse(addresses)
}
