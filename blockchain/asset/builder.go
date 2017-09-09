package asset

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"time"

	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	//"chain/database/pg"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/log"
//	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

func (reg *Registry) NewIssueAction(assetAmount bc.AssetAmount, referenceData chainjson.Map) txbuilder.Action {
	return &issueAction{
		assets:        reg,
		AssetAmount:   assetAmount,
		ReferenceData: referenceData,
	}
}

func (reg *Registry) DecodeIssueAction(data []byte) (txbuilder.Action, error) {
	a := &issueAction{assets: reg}
	err := json.Unmarshal(data, a)
	return a, err
}

type issueAction struct {
	assets *Registry
	bc.AssetAmount
	ReferenceData chainjson.Map `json:"reference_data"`
}

func (a *issueAction) Build(ctx context.Context, builder *txbuilder.TemplateBuilder) error {
	if a.AssetId.IsZero() {
		return txbuilder.MissingFieldsError("asset_id")
		return nil
	}

	asset, err := a.assets.findByID(ctx, *a.AssetId)
/*	if errors.Root(err) == pg.ErrUserInputNotFound {
		err = errors.WithDetailf(err, "missing asset with ID %x", a.AssetId.Bytes())
	}
*/	if err != nil {
		return err
	}

	var nonce [8]byte
	_, err = rand.Read(nonce[:])
	if err != nil {
		return err
	}

	assetdef := asset.RawDefinition()

	txin := legacy.NewIssuanceInput(nonce[:], a.Amount, a.ReferenceData, asset.InitialBlockHash, asset.IssuanceProgram, nil, assetdef)


	tplIn := &txbuilder.SigningInstruction{}
	path := signers.Path(asset.Signer, signers.AssetKeySpace)
	tplIn.AddWitnessKeys(asset.Signer.XPubs, path, asset.Signer.Quorum)

	log.Printf(ctx, "txin:%v\n", txin)
	log.Printf(ctx, "tplIn:%v\n", tplIn)
	builder.RestrictMinTime(time.Now())
	return builder.AddInput(txin, tplIn)
}

