package asset

import (
	"encoding/json"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/signers"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/protocol/vm/vmutil"
)

func isValidJSON(b []byte) bool {
	var v interface{}
	err := json.Unmarshal(b, &v)
	return err == nil
}

//Annotated annotate the asset
func Annotated(a *Asset) (*query.AnnotatedAsset, error) {
	jsonDefinition := json.RawMessage(`{}`)

	// a.RawDefinitionByte is the asset definition as it appears on the
	// blockchain, so it's untrusted and may not be valid json.
	if isValidJSON(a.RawDefinitionByte) {
		jsonDefinition = json.RawMessage(a.RawDefinitionByte)
	}

	aa := &query.AnnotatedAsset{
		ID:              a.AssetID,
		Definition:      &jsonDefinition,
		IssuanceProgram: chainjson.HexBytes(a.IssuanceProgram),
	}
	if a.Alias != nil {
		aa.Alias = *a.Alias
	}
	if a.Signer != nil {
		path := signers.GetBip0032Path(a.Signer, signers.AssetKeySpace)
		var jsonPath []chainjson.HexBytes
		for _, p := range path {
			jsonPath = append(jsonPath, p)
		}
		for _, xpub := range a.Signer.XPubs {
			derived := xpub.Derive(path)
			aa.Keys = append(aa.Keys, &query.AssetKey{
				RootXPub:            xpub,
				AssetPubkey:         derived[:],
				AssetDerivationPath: jsonPath,
			})
		}
		aa.Quorum = a.Signer.Quorum
	} else {
		pubkeys, quorum, err := vmutil.ParseP2SPMultiSigProgram(a.IssuanceProgram)
		if err == nil {
			for _, pubkey := range pubkeys {
				pubkey := pubkey
				aa.Keys = append(aa.Keys, &query.AssetKey{
					AssetPubkey: chainjson.HexBytes(pubkey[:]),
				})
			}
			aa.Quorum = quorum
		}
	}
	return aa, nil
}
