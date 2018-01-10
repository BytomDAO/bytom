package account

import (
	"encoding/json"

	"github.com/bytom/blockchain/query"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/protocol/bc"
)

const (
	//UTXOPreFix is AccountUTXOKey prefix
	UTXOPreFix = "ACU:"
)

//UTXOKey makes a account unspent outputs key to store
func UTXOKey(id bc.Hash) []byte {
	name := id.String()
	return []byte(UTXOPreFix + name)
}

var emptyJSONObject = json.RawMessage(`{}`)

//Annotated init an annotated account object
func Annotated(a *Account) (*query.AnnotatedAccount, error) {
	aa := &query.AnnotatedAccount{
		ID:     a.ID,
		Alias:  a.Alias,
		Quorum: a.Quorum,
		Tags:   &emptyJSONObject,
	}

	tags, err := json.Marshal(a.Tags)
	if err != nil {
		return nil, err
	}
	if len(tags) > 0 {
		rawTags := json.RawMessage(tags)
		aa.Tags = &rawTags
	}
	path := path(a.KeyIndex)
	if err != nil {
		return nil, err
	}
	var jsonPath []chainjson.HexBytes
	for _, p := range path {
		jsonPath = append(jsonPath, p)
	}
	for _, xpub := range a.XPubs {
		aa.Keys = append(aa.Keys, &query.AccountKey{
			AccountXPub:           xpub,
			AccountDerivationPath: jsonPath,
		})
	}
	return aa, nil
}
