package account

import (
	"github.com/bytom/bytom/blockchain/query"
	"github.com/bytom/bytom/protocol/bc"
)

const (
	//UTXOPreFix is StandardUTXOKey prefix
	UTXOPreFix = "ACU:"
	//SUTXOPrefix is ContractUTXOKey prefix
	SUTXOPrefix = "SCU:"
)

// StandardUTXOKey makes an account unspent outputs key to store
func StandardUTXOKey(id bc.Hash) []byte {
	name := id.String()
	return []byte(UTXOPreFix + name)
}

// ContractUTXOKey makes a smart contract unspent outputs key to store
func ContractUTXOKey(id bc.Hash) []byte {
	name := id.String()
	return []byte(SUTXOPrefix + name)
}

//Annotated init an annotated account object
func Annotated(a *Account) *query.AnnotatedAccount {
	return &query.AnnotatedAccount{
		ID:         a.ID,
		Alias:      a.Alias,
		Quorum:     a.Quorum,
		XPubs:      a.XPubs,
		KeyIndex:   a.KeyIndex,
		DeriveRule: a.DeriveRule,
	}
}
