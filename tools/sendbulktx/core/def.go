package core

import (
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/crypto/ed25519/chainkd"
)

type keyIns struct {
	Alias    string `json:"alias"`
	Password string `json:"password"`
}

// Reveive use while CreateReceiver
type Reveive struct {
	AccountID    string `json:"account_id"`
	AccountAlias string `json:"account_alias"`
}

type account struct {
	RootXPubs   []chainkd.XPub         `json:"root_xpubs"`
	Quorum      int                    `json:"quorum"`
	Alias       string                 `json:"alias"`
	Tags        map[string]interface{} `json:"tags"`
	AccessToken string                 `json:"access_token"`
}

type asset struct {
	RootXPubs   []chainkd.XPub         `json:"root_xpubs"`
	Quorum      int                    `json:"quorum"`
	Alias       string                 `json:"alias"`
	Tags        map[string]interface{} `json:"tags"`
	Definition  map[string]interface{} `json:"definition"`
	AccessToken string                 `json:"access_token"`
}

type signResp struct {
	Tx           *txbuilder.Template `json:"transaction"`
	SignComplete bool                `json:"sign_complete"`
}
