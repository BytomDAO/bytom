package test

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/protocol/vm/vmutil"
)

// TxGenerator used to generate new tx
type TxGenerator struct {
	Builder        *txbuilder.TemplateBuilder
	AccountManager *account.Manager
	Assets         *asset.Registry
	Hsm            *pseudohsm.HSM
}

// NewTxGenerator create a TxGenerator
func NewTxGenerator(accountManager *account.Manager, assets *asset.Registry, hsm *pseudohsm.HSM) *TxGenerator {
	return &TxGenerator{
		Builder:        txbuilder.NewBuilder(time.Now()),
		AccountManager: accountManager,
		Assets:         assets,
		Hsm:            hsm,
	}
}

// Reset reset transaction builder, used to create a new tx
func (g *TxGenerator) Reset() {
	g.Builder = txbuilder.NewBuilder(time.Now())
}

func (g *TxGenerator) createKey(alias string, auth string) error {
	_, err := g.Hsm.XCreate(alias, auth)
	return err
}

func (g *TxGenerator) getPubkey(keyAlias string) *chainkd.XPub {
	pubKeys := g.Hsm.ListKeys()
	for i, key := range pubKeys {
		if key.Alias == keyAlias {
			return &pubKeys[i].XPub
		}
	}
	return nil
}

func (g *TxGenerator) createAccount(name string, keys []string, quorum int) error {
	xpubs := []chainkd.XPub{}
	for _, alias := range keys {
		xpub := g.getPubkey(alias)
		if xpub == nil {
			return fmt.Errorf("can't find pubkey for %s", alias)
		}
		xpubs = append(xpubs, *xpub)
	}
	_, err := g.AccountManager.Create(nil, xpubs, quorum, name, nil)
	return err
}

func (g *TxGenerator) createAsset(accountAlias string, assetAlias string) (*asset.Asset, error) {
	acc, err := g.AccountManager.FindByAlias(nil, accountAlias)
	if err != nil {
		return nil, err
	}
	return g.Assets.Define(acc.XPubs, len(acc.XPubs), nil, assetAlias, nil)
}

func (g *TxGenerator) mockUtxo(accountAlias, assetAlias string, amount uint64) (*account.UTXO, error) {
	ctrlProg, err := g.createControlProgram(accountAlias)
	if err != nil {
		return nil, err
	}

	assetAmount, err := g.assetAmount(assetAlias, amount)
	if err != nil {
		return nil, err
	}
	utxo := &account.UTXO{
		OutputID:            bc.Hash{V0: 1},
		SourceID:            bc.Hash{V0: 1},
		AssetID:             *assetAmount.AssetId,
		Amount:              assetAmount.Amount,
		SourcePos:           0,
		ControlProgram:      ctrlProg.ControlProgram,
		ControlProgramIndex: ctrlProg.KeyIndex,
		AccountID:           ctrlProg.AccountID,
		Address:             ctrlProg.Address,
		ValidHeight:         0,
	}
	return utxo, nil
}

func (g *TxGenerator) assetAmount(assetAlias string, amount uint64) (*bc.AssetAmount, error) {
	if assetAlias == "BTM" {
		a := &bc.AssetAmount{
			Amount:  amount,
			AssetId: consensus.BTMAssetID,
		}
		return a, nil
	}

	asset, err := g.Assets.FindByAlias(nil, assetAlias)
	if err != nil {
		return nil, err
	}
	return &bc.AssetAmount{
		Amount:  amount,
		AssetId: &asset.AssetID,
	}, nil
}

func (g *TxGenerator) createControlProgram(accountAlias string) (*account.CtrlProgram, error) {
	acc, err := g.AccountManager.FindByAlias(nil, accountAlias)
	if err != nil {
		return nil, err
	}
	return g.AccountManager.CreateAddress(nil, acc.ID)
}

// AddSpendInput add a spend input
func (g *TxGenerator) AddSpendInput(accountAlias, assetAlias string, amount uint64) error {
	assetAmount, err := g.assetAmount(assetAlias, amount)
	if err != nil {
		return err
	}

	acc, err := g.AccountManager.FindByAlias(nil, accountAlias)
	if err != nil {
		return err
	}

	reqAction := make(map[string]interface{})
	reqAction["account_id"] = acc.ID
	reqAction["amount"] = amount
	reqAction["asset_id"] = assetAmount.AssetId.String()
	data, err := json.Marshal(reqAction)
	if err != nil {
		return err
	}

	spendAction, err := g.AccountManager.DecodeSpendAction(data)
	if err != nil {
		return err
	}
	return spendAction.Build(nil, g.Builder)
}

// AddTxInput add a tx input and signing instruction
func (g *TxGenerator) AddTxInput(txInput *types.TxInput, signInstruction *txbuilder.SigningInstruction) error {
	return g.Builder.AddInput(txInput, signInstruction)
}

// AddTxInputFromUtxo add a tx input which spent the utxo
func (g *TxGenerator) AddTxInputFromUtxo(utxo *account.UTXO, accountAlias string) error {
	acc, err := g.AccountManager.FindByAlias(nil, accountAlias)
	if err != nil {
		return err
	}

	txInput, signInst, err := account.UtxoToInputs(acc.Signer, utxo)
	if err != nil {
		return err
	}
	return g.AddTxInput(txInput, signInst)
}

// AddIssuanceInput add a issue input
func (g *TxGenerator) AddIssuanceInput(assetAlias string, amount uint64) error {
	asset, err := g.Assets.FindByAlias(nil, assetAlias)
	if err != nil {
		return err
	}

	var nonce [8]byte
	_, err = rand.Read(nonce[:])
	if err != nil {
		return err
	}
	issuanceInput := types.NewIssuanceInput(nonce[:], amount, asset.IssuanceProgram, nil, asset.RawDefinitionByte)
	signInstruction := &txbuilder.SigningInstruction{}
	path := signers.Path(asset.Signer, signers.AssetKeySpace)
	signInstruction.AddRawWitnessKeys(asset.Signer.XPubs, path, asset.Signer.Quorum)
	g.Builder.RestrictMinTime(time.Now())
	return g.Builder.AddInput(issuanceInput, signInstruction)
}

// AddTxOutput add a tx output
func (g *TxGenerator) AddTxOutput(accountAlias, assetAlias string, amount uint64) error {
	assetAmount, err := g.assetAmount(assetAlias, uint64(amount))
	if err != nil {
		return err
	}
	controlProgram, err := g.createControlProgram(accountAlias)
	if err != nil {
		return err
	}
	out := types.NewTxOutput(*assetAmount.AssetId, assetAmount.Amount, controlProgram.ControlProgram)
	return g.Builder.AddOutput(out)
}

// AddRetirement add a retirement output
func (g *TxGenerator) AddRetirement(assetAlias string, amount uint64) error {
	assetAmount, err := g.assetAmount(assetAlias, uint64(amount))
	if err != nil {
		return err
	}
	retirementProgram := []byte{byte(vm.OP_FAIL)}
	out := types.NewTxOutput(*assetAmount.AssetId, assetAmount.Amount, retirementProgram)
	return g.Builder.AddOutput(out)
}

// Sign used to sign tx
func (g *TxGenerator) Sign(passwords []string) (*types.Tx, error) {
	tpl, _, err := g.Builder.Build()
	if err != nil {
		return nil, err
	}

	txSerialized, err := tpl.Transaction.MarshalText()
	if err != nil {
		return nil, err
	}

	tpl.Transaction.Tx.SerializedSize = uint64(len(txSerialized))
	tpl.Transaction.TxData.SerializedSize = uint64(len(txSerialized))
	for _, password := range passwords {
		_, err = MockSign(tpl, g.Hsm, password)
		if err != nil {
			return nil, err
		}
	}
	return tpl.Transaction, nil
}

func txFee(tx *types.Tx) uint64 {
	if len(tx.Inputs) == 1 && tx.Inputs[0].InputType() == types.CoinbaseInputType {
		return 0
	}

	inputSum := uint64(0)
	outputSum := uint64(0)
	for _, input := range tx.Inputs {
		if input.AssetID() == *consensus.BTMAssetID {
			inputSum += input.Amount()
		}
	}

	for _, output := range tx.Outputs {
		if *output.AssetId == *consensus.BTMAssetID {
			outputSum += output.Amount
		}
	}
	return inputSum - outputSum
}

// CreateSpendInput create SpendInput which spent the output from tx
func CreateSpendInput(tx *types.Tx, outputIndex uint64) (*types.SpendInput, error) {
	outputID := tx.ResultIds[outputIndex]
	output, ok := tx.Entries[*outputID].(*bc.Output)
	if !ok {
		return nil, fmt.Errorf("retirement can't be spent")
	}

	sc := types.SpendCommitment{
		AssetAmount:    *output.Source.Value,
		SourceID:       *output.Source.Ref,
		SourcePosition: output.Ordinal,
		VMVersion:      vmVersion,
		ControlProgram: output.ControlProgram.Code,
	}
	return &types.SpendInput{
		SpendCommitment: sc,
	}, nil
}

// SignInstructionFor read CtrlProgram from db, construct SignInstruction for SpendInput
func SignInstructionFor(input *types.SpendInput, db db.DB, signer *signers.Signer) (*txbuilder.SigningInstruction, error) {
	cp := account.CtrlProgram{}
	var hash [32]byte
	sha3pool.Sum256(hash[:], input.ControlProgram)
	bytes := db.Get(account.CPKey(hash))
	if bytes == nil {
		return nil, fmt.Errorf("can't find CtrlProgram for the SpendInput")
	}

	err := json.Unmarshal(bytes, &cp)
	if err != nil {
		return nil, err
	}

	sigInst := &txbuilder.SigningInstruction{}
	if signer == nil {
		return sigInst, nil
	}

	// FIXME: code duplicate with account/builder.go
	path := signers.Path(signer, signers.AccountKeySpace, cp.KeyIndex)
	if cp.Address == "" {
		sigInst.AddWitnessKeys(signer.XPubs, path, signer.Quorum)
		return sigInst, nil
	}

	address, err := common.DecodeAddress(cp.Address, &consensus.MainNetParams)
	if err != nil {
		return nil, err
	}

	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		sigInst.AddRawWitnessKeys(signer.XPubs, path, signer.Quorum)
		derivedXPubs := chainkd.DeriveXPubs(signer.XPubs, path)
		derivedPK := derivedXPubs[0].PublicKey()
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness([]byte(derivedPK)))

	case *common.AddressWitnessScriptHash:
		sigInst.AddRawWitnessKeys(signer.XPubs, path, signer.Quorum)
		path := signers.Path(signer, signers.AccountKeySpace, cp.KeyIndex)
		derivedXPubs := chainkd.DeriveXPubs(signer.XPubs, path)
		derivedPKs := chainkd.XPubKeys(derivedXPubs)
		script, err := vmutil.P2SPMultiSigProgram(derivedPKs, signer.Quorum)
		if err != nil {
			return nil, err
		}
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness(script))

	default:
		return nil, errors.New("unsupport address type")
	}

	return sigInst, nil
}

// CreateCoinbaseTx create coinbase tx at block height
func CreateCoinbaseTx(controlProgram []byte, height, txsFee uint64) (*types.Tx, error) {
	coinbaseValue := consensus.BlockSubsidy(height) + txsFee
	builder := txbuilder.NewBuilder(time.Now())
	if err := builder.AddInput(types.NewCoinbaseInput([]byte(string(height))), &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}
	if err := builder.AddOutput(types.NewTxOutput(*consensus.BTMAssetID, coinbaseValue, controlProgram)); err != nil {
		return nil, err
	}

	tpl, _, err := builder.Build()
	if err != nil {
		return nil, err
	}

	txSerialized, err := tpl.Transaction.MarshalText()
	if err != nil {
		return nil, err
	}

	tpl.Transaction.Tx.SerializedSize = uint64(len(txSerialized))
	tpl.Transaction.TxData.SerializedSize = uint64(len(txSerialized))
	return tpl.Transaction, nil
}

// CreateTxFromTx create a tx spent the output in outputIndex at baseTx
func CreateTxFromTx(baseTx *types.Tx, outputIndex uint64, outputAmount uint64, ctrlProgram []byte) (*types.Tx, error) {
	spendInput, err := CreateSpendInput(baseTx, outputIndex)
	if err != nil {
		return nil, err
	}

	txInput := &types.TxInput{
		AssetVersion: assetVersion,
		TypedInput:   spendInput,
	}
	output := types.NewTxOutput(*consensus.BTMAssetID, outputAmount, ctrlProgram)
	builder := txbuilder.NewBuilder(time.Now())
	builder.AddInput(txInput, &txbuilder.SigningInstruction{})
	builder.AddOutput(output)

	tpl, _, err := builder.Build()
	if err != nil {
		return nil, err
	}

	txSerialized, err := tpl.Transaction.MarshalText()
	if err != nil {
		return nil, err
	}

	tpl.Transaction.Tx.SerializedSize = uint64(len(txSerialized))
	tpl.Transaction.TxData.SerializedSize = uint64(len(txSerialized))
	return tpl.Transaction, nil
}
