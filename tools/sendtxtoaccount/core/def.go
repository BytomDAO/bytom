package core

import (
	"github.com/bytom/blockchain/txbuilder"
)

type signResp struct {
	Tx           *txbuilder.Template `json:"transaction"`
	SignComplete bool                `json:"sign_complete"`
}

type accountInfo struct {
	address string
	amount  int
}
