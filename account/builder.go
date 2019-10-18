package account

import (
	"context"
	"encoding/json"

	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm/vmutil"
)

//DecodeSpendAction unmarshal JSON-encoded data of spend action
func (m *Manager) DecodeSpendAction(data []byte) (txbuilder.Action, error) {
	a := &spendAction{accounts: m}
	return a, json.Unmarshal(data, a)
}

type spendAction struct {
	accounts *Manager
	bc.AssetAmount
	AccountID      string `json:"account_id"`
	UseUnconfirmed bool   `json:"use_unconfirmed"`
}

func (a *spendAction) ActionType() string {
	return "spend_account"
}

// MergeSpendAction merge common assetID and accountID spend action
func MergeSpendAction(actions []txbuilder.Action) []txbuilder.Action {
	resultActions := []txbuilder.Action{}
	spendActionMap := make(map[string]*spendAction)

	for _, act := range actions {
		switch act := act.(type) {
		case *spendAction:
			actionKey := act.AssetId.String() + act.AccountID
			if tmpAct, ok := spendActionMap[actionKey]; ok {
				tmpAct.Amount += act.Amount
				tmpAct.UseUnconfirmed = tmpAct.UseUnconfirmed || act.UseUnconfirmed
			} else {
				spendActionMap[actionKey] = act
				resultActions = append(resultActions, act)
			}
		default:
			resultActions = append(resultActions, act)
		}
	}
	return resultActions
}

//calcMergeGas calculate the gas required that n utxos are merged into one
func calcMergeGas(num int) uint64 {
	gas := uint64(0)
	for num > 1 {
		gas += txbuilder.ChainTxMergeGas
		num -= txbuilder.ChainTxUtxoNum - 1
	}
	return gas
}

func (m *Manager) reserveBtmUtxoChain(builder *txbuilder.TemplateBuilder, accountID string, amount uint64, useUnconfirmed bool) ([]*UTXO, error) {
	reservedAmount := uint64(0)
	utxos := []*UTXO{}
	for gasAmount := uint64(0); reservedAmount < gasAmount+amount; gasAmount = calcMergeGas(len(utxos)) {
		reserveAmount := amount + gasAmount - reservedAmount
		res, err := m.utxoKeeper.Reserve(accountID, consensus.BTMAssetID, reserveAmount, useUnconfirmed, builder.MaxTime())
		if err != nil {
			return nil, err
		}

		builder.OnRollback(func() { m.utxoKeeper.Cancel(res.id) })
		reservedAmount += reserveAmount + res.change
		utxos = append(utxos, res.utxos[:]...)
	}
	return utxos, nil
}

func (m *Manager) buildBtmTxChain(utxos []*UTXO, signer *signers.Signer) ([]*txbuilder.Template, *UTXO, error) {
	if len(utxos) == 0 {
		return nil, nil, errors.New("mergeSpendActionUTXO utxos num 0")
	}

	tpls := []*txbuilder.Template{}
	if len(utxos) == 1 {
		return tpls, utxos[len(utxos)-1], nil
	}

	acp, err := m.GetLocalCtrlProgramByAddress(utxos[0].Address)
	if err != nil {
		return nil, nil, err
	}

	buildAmount := uint64(0)
	builder := &txbuilder.TemplateBuilder{}
	for index := 0; index < len(utxos); index++ {
		input, sigInst, err := UtxoToInputs(signer, utxos[index])
		if err != nil {
			return nil, nil, err
		}

		if err = builder.AddInput(input, sigInst); err != nil {
			return nil, nil, err
		}

		buildAmount += input.Amount()
		if builder.InputCount() != txbuilder.ChainTxUtxoNum && index != len(utxos)-1 {
			continue
		}

		outAmount := buildAmount - txbuilder.ChainTxMergeGas
		output := types.NewTxOutput(*consensus.BTMAssetID, outAmount, acp.ControlProgram)
		if err := builder.AddOutput(output); err != nil {
			return nil, nil, err
		}

		tpl, _, err := builder.Build()
		if err != nil {
			return nil, nil, err
		}

		bcOut, err := tpl.Transaction.Output(*tpl.Transaction.ResultIds[0])
		if err != nil {
			return nil, nil, err
		}

		utxos = append(utxos, &UTXO{
			OutputID:            *tpl.Transaction.ResultIds[0],
			AssetID:             *consensus.BTMAssetID,
			Amount:              outAmount,
			ControlProgram:      acp.ControlProgram,
			SourceID:            *bcOut.Source.Ref,
			SourcePos:           bcOut.Source.Position,
			ControlProgramIndex: acp.KeyIndex,
			Address:             acp.Address,
			Change:              acp.Change,
		})

		tpls = append(tpls, tpl)
		buildAmount = 0
		builder = &txbuilder.TemplateBuilder{}
		if index == len(utxos)-2 {
			break
		}
	}
	return tpls, utxos[len(utxos)-1], nil
}

// SpendAccountChain build the spend action with auto merge utxo function
func SpendAccountChain(ctx context.Context, builder *txbuilder.TemplateBuilder, action txbuilder.Action) ([]*txbuilder.Template, error) {
	act, ok := action.(*spendAction)
	if !ok {
		return nil, errors.New("fail to convert the spend action")
	}
	if *act.AssetId != *consensus.BTMAssetID {
		return nil, errors.New("spend chain action only support BTM")
	}

	utxos, err := act.accounts.reserveBtmUtxoChain(builder, act.AccountID, act.Amount, act.UseUnconfirmed)
	if err != nil {
		return nil, err
	}

	acct, err := act.accounts.FindByID(act.AccountID)
	if err != nil {
		return nil, err
	}

	tpls, utxo, err := act.accounts.buildBtmTxChain(utxos, acct.Signer)
	if err != nil {
		return nil, err
	}

	input, sigInst, err := UtxoToInputs(acct.Signer, utxo)
	if err != nil {
		return nil, err
	}

	if err := builder.AddInput(input, sigInst); err != nil {
		return nil, err
	}

	if utxo.Amount > act.Amount {
		if err = builder.AddOutput(types.NewTxOutput(*consensus.BTMAssetID, utxo.Amount-act.Amount, utxo.ControlProgram)); err != nil {
			return nil, errors.Wrap(err, "adding change output")
		}
	}
	return tpls, nil
}

func (a *spendAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var missing []string
	if a.AccountID == "" {
		missing = append(missing, "account_id")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.AssetAmount.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	acct, err := a.accounts.FindByID(a.AccountID)
	if err != nil {
		return errors.Wrap(err, "get account info")
	}

	res, err := a.accounts.utxoKeeper.Reserve(a.AccountID, a.AssetId, a.Amount, a.UseUnconfirmed, b.MaxTime())
	if err != nil {
		return errors.Wrap(err, "reserving utxos")
	}

	// Cancel the reservation if the build gets rolled back.
	b.OnRollback(func() { a.accounts.utxoKeeper.Cancel(res.id) })
	for _, r := range res.utxos {
		txInput, sigInst, err := UtxoToInputs(acct.Signer, r)
		if err != nil {
			return errors.Wrap(err, "creating inputs")
		}

		if err = b.AddInput(txInput, sigInst); err != nil {
			return errors.Wrap(err, "adding inputs")
		}
	}

	if res.change > 0 {
		acp, err := a.accounts.CreateAddress(a.AccountID, true)
		if err != nil {
			return errors.Wrap(err, "creating control program")
		}

		// Don't insert the control program until callbacks are executed.
		a.accounts.insertControlProgramDelayed(b, acp)
		if err = b.AddOutput(types.NewTxOutput(*a.AssetId, res.change, acp.ControlProgram)); err != nil {
			return errors.Wrap(err, "adding change output")
		}
	}
	return nil
}

//DecodeSpendUTXOAction unmarshal JSON-encoded data of spend utxo action
func (m *Manager) DecodeSpendUTXOAction(data []byte) (txbuilder.Action, error) {
	a := &spendUTXOAction{accounts: m}
	return a, json.Unmarshal(data, a)
}

type spendUTXOAction struct {
	accounts       *Manager
	OutputID       *bc.Hash                     `json:"output_id"`
	UseUnconfirmed bool                         `json:"use_unconfirmed"`
	Arguments      []txbuilder.ContractArgument `json:"arguments"`
}

func (a *spendUTXOAction) ActionType() string {
	return "spend_account_unspent_output"
}

func (a *spendUTXOAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	if a.OutputID == nil {
		return txbuilder.MissingFieldsError("output_id")
	}

	res, err := a.accounts.utxoKeeper.ReserveParticular(*a.OutputID, a.UseUnconfirmed, b.MaxTime())
	if err != nil {
		return err
	}

	b.OnRollback(func() { a.accounts.utxoKeeper.Cancel(res.id) })
	var accountSigner *signers.Signer
	if len(res.utxos[0].AccountID) != 0 {
		account, err := a.accounts.FindByID(res.utxos[0].AccountID)
		if err != nil {
			return err
		}

		accountSigner = account.Signer
	}

	txInput, sigInst, err := UtxoToInputs(accountSigner, res.utxos[0])
	if err != nil {
		return err
	}

	if a.Arguments == nil {
		return b.AddInput(txInput, sigInst)
	}

	sigInst = &txbuilder.SigningInstruction{}
	if err := txbuilder.AddContractArgs(sigInst, a.Arguments); err != nil {
		return err
	}

	return b.AddInput(txInput, sigInst)
}

// UtxoToInputs convert an utxo to the txinput
func UtxoToInputs(signer *signers.Signer, u *UTXO) (*types.TxInput, *txbuilder.SigningInstruction, error) {
	txInput := types.NewSpendInput(nil, u.SourceID, u.AssetID, u.Amount, u.SourcePos, u.ControlProgram)
	sigInst := &txbuilder.SigningInstruction{}
	if signer == nil {
		return txInput, sigInst, nil
	}

	path, err := signers.Path(signer, signers.AccountKeySpace, u.Change, u.ControlProgramIndex)
	if err != nil {
		return nil, nil, err
	}
	if u.Address == "" {
		sigInst.AddWitnessKeys(signer.XPubs, path, signer.Quorum)
		return txInput, sigInst, nil
	}

	address, err := common.DecodeAddress(u.Address, &consensus.ActiveNetParams)
	if err != nil {
		return nil, nil, err
	}

	sigInst.AddRawWitnessKeys(signer.XPubs, path, signer.Quorum)
	derivedXPubs := chainkd.DeriveXPubs(signer.XPubs, path)

	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		derivedPK := derivedXPubs[0].PublicKey()
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness([]byte(derivedPK)))

	case *common.AddressWitnessScriptHash:
		derivedPKs := chainkd.XPubKeys(derivedXPubs)
		script, err := vmutil.P2SPMultiSigProgram(derivedPKs, signer.Quorum)
		if err != nil {
			return nil, nil, err
		}
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness(script))

	default:
		return nil, nil, errors.New("unsupport address type")
	}

	return txInput, sigInst, nil
}

// insertControlProgramDelayed takes a template builder and an account
// control program that hasn't been inserted to the database yet. It
// registers callbacks on the TemplateBuilder so that all of the template's
// account control programs are batch inserted if building the rest of
// the template is successful.
func (m *Manager) insertControlProgramDelayed(b *txbuilder.TemplateBuilder, acp *CtrlProgram) {
	m.delayedACPsMu.Lock()
	m.delayedACPs[b] = append(m.delayedACPs[b], acp)
	m.delayedACPsMu.Unlock()

	b.OnRollback(func() {
		m.delayedACPsMu.Lock()
		delete(m.delayedACPs, b)
		m.delayedACPsMu.Unlock()
	})
	b.OnBuild(func() error {
		m.delayedACPsMu.Lock()
		acps := m.delayedACPs[b]
		delete(m.delayedACPs, b)
		m.delayedACPsMu.Unlock()

		// Insert all of the account control programs at once.
		if len(acps) == 0 {
			return nil
		}
		return m.SaveControlPrograms(acps...)
	})
}
