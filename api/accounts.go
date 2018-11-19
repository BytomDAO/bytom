package api

import (
	"context"
	"encoding/hex"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol/vm/vmutil"
)

// POST /create-account
func (a *API) createAccount(ctx context.Context, ins struct {
	RootXPubs []chainkd.XPub `json:"root_xpubs"`
	Quorum    int            `json:"quorum"`
	Alias     string         `json:"alias"`
}) Response {
	acc, err := a.wallet.AccountMgr.Create(ins.RootXPubs, ins.Quorum, ins.Alias, signers.BIP0044)
	if err != nil {
		return NewErrorResponse(err)
	}

	annotatedAccount := account.Annotated(acc)
	log.WithField("account ID", annotatedAccount.ID).Info("Created account")

	return NewSuccessResponse(annotatedAccount)
}

// POST update-account-alias
func (a *API) updateAccountAlias(ctx context.Context, ins struct {
	AccountID    string `json:"account_id"`
	AccountAlias string `json:"account_alias"`
	NewAlias     string `json:"new_alias"`
}) Response {
	accountID := ins.AccountID
	if ins.AccountAlias != "" {
		foundAccount, err := a.wallet.AccountMgr.FindByAlias(ins.AccountAlias)
		if err != nil {
			return NewErrorResponse(err)
		}
		accountID = foundAccount.ID
	}
	if err := a.wallet.UpdateAccountAlias(accountID, ins.NewAlias); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(nil)
}

// AccountInfo is request struct for deleteAccount
type AccountInfo struct {
	Info string `json:"account_info"`
}

// POST /delete-account
func (a *API) deleteAccount(ctx context.Context, filter struct {
	AccountID    string `json:"account_id"`
	AccountAlias string `json:"account_alias"`
}) Response {
	accountID := filter.AccountID
	if filter.AccountAlias != "" {
		acc, err := a.wallet.AccountMgr.FindByAlias(filter.AccountAlias)
		if err != nil {
			return NewErrorResponse(err)
		}
		accountID = acc.ID
	}
	if err := a.wallet.DeleteAccount(accountID); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(nil)
}

type validateAddressResp struct {
	Valid   bool `json:"valid"`
	IsLocal bool `json:"is_local"`
}

// POST /validate-address
func (a *API) validateAddress(ctx context.Context, ins struct {
	Address string `json:"address"`
}) Response {
	resp := &validateAddressResp{
		Valid:   false,
		IsLocal: false,
	}
	address, err := common.DecodeAddress(ins.Address, &consensus.ActiveNetParams)
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

	resp.Valid = true
	resp.IsLocal = a.wallet.AccountMgr.IsLocalControlProgram(program)
	return NewSuccessResponse(resp)
}

type addressResp struct {
	AccountAlias   string `json:"account_alias"`
	AccountID      string `json:"account_id"`
	Address        string `json:"address"`
	ControlProgram string `json:"control_program"`
	Change         bool   `json:"change"`
	KeyIndex       uint64 `json:"key_index"`
}

// SortByIndex implements sort.Interface for addressResp slices
type SortByIndex []addressResp

func (a SortByIndex) Len() int           { return len(a) }
func (a SortByIndex) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortByIndex) Less(i, j int) bool { return a[i].KeyIndex < a[j].KeyIndex }

func (a *API) listAddresses(ctx context.Context, ins struct {
	AccountID    string `json:"account_id"`
	AccountAlias string `json:"account_alias"`
	From         uint   `json:"from"`
	Count        uint   `json:"count"`
}) Response {
	accountID := ins.AccountID
	var target *account.Account
	if ins.AccountAlias != "" {
		acc, err := a.wallet.AccountMgr.FindByAlias(ins.AccountAlias)
		if err != nil {
			return NewErrorResponse(err)
		}
		target = acc
	} else {
		acc, err := a.wallet.AccountMgr.FindByID(accountID)
		if err != nil {
			return NewErrorResponse(err)
		}
		target = acc
	}

	cps, err := a.wallet.AccountMgr.ListControlProgram()
	if err != nil {
		return NewErrorResponse(err)
	}

	addresses := []addressResp{}
	for _, cp := range cps {
		if cp.Address == "" || cp.AccountID != target.ID {
			continue
		}
		addresses = append(addresses, addressResp{
			AccountAlias:   target.Alias,
			AccountID:      cp.AccountID,
			Address:        cp.Address,
			ControlProgram: hex.EncodeToString(cp.ControlProgram),
			Change:         cp.Change,
			KeyIndex:       cp.KeyIndex,
		})
	}

	// sort AddressResp by KeyIndex
	sort.Sort(SortByIndex(addresses))
	start, end := getPageRange(len(addresses), ins.From, ins.Count)
	return NewSuccessResponse(addresses[start:end])
}

type minigAddressResp struct {
	MiningAddress string `json:"mining_address"`
}

func (a *API) getMiningAddress(ctx context.Context) Response {
	miningAddress, err := a.wallet.AccountMgr.GetMiningAddress()
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(minigAddressResp{
		MiningAddress: miningAddress,
	})
}

// POST /set-mining-address
func (a *API) setMiningAddress(ctx context.Context, in struct {
	MiningAddress string `json:"mining_address"`
}) Response {
	miningAddress, err := a.wallet.AccountMgr.SetMiningAddress(in.MiningAddress)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(minigAddressResp{
		MiningAddress: miningAddress,
	})
}
