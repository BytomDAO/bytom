package asset

import (
	"encoding/json"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/signers"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/protocol/vm"
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

	annotatedAsset := &query.AnnotatedAsset{
		ID:                a.AssetID,
		Alias:             *a.Alias,
		RawDefinitionByte: a.RawDefinitionByte,
		Definition:        &jsonDefinition,
		IssuanceProgram:   chainjson.HexBytes(a.IssuanceProgram),
	}

	insts, err := vm.ParseProgram(a.IssuanceProgram)
	if err != nil {
		return nil, err
	}

	for i, inst := range insts {
		if i-1 >= 0 && insts[i-1].IsPushdata() && inst.Op == vm.OP_BLOCKHEIGHT {
			annotatedAsset.LimitHeight, err = vm.AsInt64(insts[i-1].Data)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if a.Signer != nil {
		path := signers.GetBip0032Path(a.Signer, signers.AssetKeySpace)
		var jsonPath []chainjson.HexBytes
		for _, p := range path {
			jsonPath = append(jsonPath, p)
		}
		for _, xpub := range a.Signer.XPubs {
			derived := xpub.Derive(path)
			annotatedAsset.Keys = append(annotatedAsset.Keys, &query.AssetKey{
				RootXPub:            xpub,
				AssetPubkey:         derived[:],
				AssetDerivationPath: jsonPath,
			})
		}
		annotatedAsset.Quorum = a.Signer.Quorum
	}
	return annotatedAsset, nil
}
