package asset

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

//NewIssueAction create a new asset issue action
func (reg *Registry) NewIssueAction(assetAmount bc.AssetAmount) txbuilder.Action {
	return &issueAction{
		assets:        reg,
		AssetAmount:   assetAmount,
	}
}

//DecodeIssueAction unmarshal JSON-encoded data of asset issue action
func (reg *Registry) DecodeIssueAction(data []byte) (txbuilder.Action, error) {
	a := &issueAction{assets: reg}
	err := json.Unmarshal(data, a)
	return a, err
}

type issueAction struct {
	assets *Registry
	bc.AssetAmount
}

func (a *issueAction) Build(ctx context.Context, builder *txbuilder.TemplateBuilder) error {
	if a.AssetId.IsZero() {
		return txbuilder.MissingFieldsError("asset_id")
	}

	asset, err := a.assets.FindByID(ctx, a.AssetId)
	if err != nil {
		return err
	}

	var nonce [8]byte
	_, err = rand.Read(nonce[:])
	if err != nil {
		return err
	}

	assetDef := asset.RawDefinitionByte

	txin := legacy.NewIssuanceInput(nonce[:], a.Amount, asset.InitialBlockHash, asset.IssuanceProgram, nil, assetDef)

	tplIn := &txbuilder.SigningInstruction{}
	path := signers.Path(asset.Signer, signers.AssetKeySpace)
	tplIn.AddRawWitnessKeys(asset.Signer.XPubs, path, asset.Signer.Quorum)

	log.Info("Issue action build")
	builder.RestrictMinTime(time.Now())
	return builder.AddInput(txin, tplIn)
}
