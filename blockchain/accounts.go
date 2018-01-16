package blockchain

import (
	"context"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/crypto/ed25519/chainkd"

	log "github.com/sirupsen/logrus"
)

// POST /create-account
func (bcr *BlockchainReactor) createAccount(ctx context.Context, ins struct {
	RootXPubs []chainkd.XPub         `json:"root_xpubs"`
	Quorum    int                    `json:"quorum"`
	Alias     string                 `json:"alias"`
	Tags      map[string]interface{} `json:"tags"`
}) Response {
	acc, err := bcr.accounts.Create(ins.RootXPubs, ins.Quorum, ins.Alias, ins.Tags)
	if err != nil {
		return NewErrorResponse(err)
	}

	annotatedAccount, err := account.Annotated(acc)
	if err != nil {
		return NewErrorResponse(err)
	}

	log.WithField("account ID", annotatedAccount.ID).Info("Created account")

	return NewSuccessResponse(annotatedAccount)
}

// POST /update-account-tags
func (bcr *BlockchainReactor) updateAccountTags(ctx context.Context, updateTag struct {
	AccountInfo string                 `json:"account_info"`
	Tags        map[string]interface{} `json:"tags"`
}) Response {

	err := bcr.accounts.UpdateTags(nil, updateTag.AccountInfo, updateTag.Tags)
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(nil)
}

//
// POST /delete-account
func (bcr *BlockchainReactor) deleteAccount(ctx context.Context, in struct {
	AccountInfo string `json:"account_info"`
}) Response {
	if err := bcr.accounts.DeleteAccount(in); err != nil {
		return resWrapper(nil, err)
	}
	return resWrapper(nil)
}
