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

	// ClientToken is the application's unique token for the account. Every account
	// should have a unique client token. The client token is used to ensure
	// idempotency of create account requests. Duplicate create account requests
	// with the same client_token will only create one account.
	AccessToken string `json:"access_token"`
}) Response {
	acc, err := bcr.accounts.Create(nil, ins.RootXPubs, ins.Quorum, ins.Alias, ins.Tags, ins.AccessToken)
	if err != nil {
		return resWrapper(nil, err)
	}

	annotatedAccount, err := account.Annotated(acc)
	if err != nil {
		return resWrapper(nil, err)
	}

	log.WithField("account ID", annotatedAccount.ID).Info("Created account")

	return resWrapper(annotatedAccount)
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
