package asset

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/signers"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/vm/vmutil"
)

func Annotated(a *Asset) (*query.AnnotatedAsset, error) {
	jsonTags := json.RawMessage(`{}`)
	jsonDefinition := json.RawMessage(`{}`)

	// a.RawDefinition is the asset definition as it appears on the
	// blockchain, so it's untrusted and may not be valid json.
	if query.IsValidJSON(a.RawDefinition()) {
		jsonDefinition = json.RawMessage(a.RawDefinition())
	}

	if a.Tags != nil {
		b, err := json.Marshal(a.Tags)
		if err != nil {
			return nil, err
		}
		jsonTags = b
	}

	aa := &query.AnnotatedAsset{
		ID:              a.AssetID,
		Definition:      &jsonDefinition,
		Tags:            &jsonTags,
		IssuanceProgram: chainjson.HexBytes(a.IssuanceProgram),
	}
	if a.Alias != nil {
		aa.Alias = *a.Alias
	}
	if a.Signer != nil {
		path := signers.Path(a.Signer, signers.AssetKeySpace)
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
		aa.IsLocal = true
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

// IndexAssets is run on every block and indexes all non-local assets.
func (reg *Registry) IndexAssets(b *legacy.Block) {

	var err error
	asset := Asset{}
	rawSaveAsset := make([]byte, 0)
	seen := make(map[bc.AssetID]bool)
	storeBatch := reg.db.NewBatch()

	for _, tx := range b.Transactions {
		for _, in := range tx.Inputs {
			if !in.IsIssuance() {
				continue
			}
			assetID := in.AssetID()
			if seen[assetID] {
				continue
			}
			inputIssue, ok := in.TypedInput.(*legacy.IssuanceInput)
			if !ok {
				continue
			}

			seen[assetID] = true

			if rawAsset := reg.db.Get([]byte(assetID.String())); rawAsset == nil {
				asset.RawDefinitionByte = inputIssue.AssetDefinition
				asset.AssetID = assetID
				asset.VMVersion = inputIssue.VMVersion
				asset.IssuanceProgram = in.IssuanceProgram()
				asset.BlockHeight = b.Height
				asset.InitialBlockHash = reg.initialBlockHash
			} else {
				if err = json.Unmarshal(rawAsset, &asset); err != nil {
					log.WithField("AssetID", assetID.String()).Warn("failed unmarshal saved asset")
					continue
				}
				//update block height which created at
				if asset.BlockHeight != 0 {
					continue
				}
				asset.BlockHeight = b.Height
			}

			rawSaveAsset, err = json.Marshal(&asset)
			if err != nil {
				log.WithField("AssetID", assetID.String()).Warn("failed marshal to save asset")
				continue
			}

			storeBatch.Set([]byte(assetID.String()), rawSaveAsset)

		}
	}

	storeBatch.Write()
}
