package account

import (
	"context"
	"encoding/json"
	"time"

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

//MaxUTXONum maximum utxo quantity of an asset in a transaction
const TxMaxInputUTXONum = 10

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

// CheckAssetType check actions asset type
func CheckActionsAssetType(actions []txbuilder.Action, assetType *bc.AssetID) bool {
	for _, act := range actions {
		switch act := act.(type) {
		case *spendAction:
			if *act.AssetId != *assetType {
				return false
			}
		default:
		}
	}
	return true
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

func mergeSpendActionUTXO(act *spendAction, preTxTemplate *txbuilder.Template, txTemplates map[string][]*txbuilder.Template, txBuilders map[string][]*txbuilder.TemplateBuilder, maxTime time.Time, timeRange uint64) error {
	newActions := []txbuilder.Action{}
	utxos, err := GetUTXO(act, preTxTemplate)
	if err != nil {
		return err
	}

	if len(utxos) > TxMaxInputUTXONum {
		amount := uint64(0)
		for i := 0; i < TxMaxInputUTXONum; i++ {
			amount += utxos[i].Amount
		}
		spendAct := new(spendAction)
		spendAct.accounts = act.accounts
		spendAct.Amount = amount
		spendAct.AssetId = act.AssetId
		spendAct.AccountID = act.AccountID
		spendAct.UseUnconfirmed = act.UseUnconfirmed
		newActions = append(newActions, spendAct)
		controlAct := txbuilder.NewControlAddressAction()
		controlAct.Amount = amount - 10000000
		controlAct.AssetId = act.AssetId
		acp, err := act.accounts.CreateAddress(act.AccountID, false)
		if err != nil {
			return err
		}
		controlAct.Address = acp.Address
		newActions = append(newActions, controlAct)

		tpl, builder, err := txbuilder.Build(nil, nil, newActions, txTemplates, maxTime, timeRange)
		if errors.Root(err) == txbuilder.ErrAction {
			// append each of the inner errors contained in the data.
			var Errs string
			var rootErr error
			for i, innerErr := range errors.Data(err)["actions"].([]error) {
				if i == 0 {
					rootErr = errors.Root(innerErr)
				}
				Errs = Errs + innerErr.Error()
			}
			err = errors.WithDetail(rootErr, Errs)
		}
		if err != nil {
			return err
		}

		// ensure null is never returned for signing instructions
		if tpl.SigningInstructions == nil {
			tpl.SigningInstructions = []*txbuilder.SigningInstruction{}
		}
		key := actTemplatesKey(act.AccountID, act.AssetId)
		tpls, ok := txTemplates[key]
		if !ok {
			txTemplates[key] = []*txbuilder.Template{tpl}
		} else {
			tpls = append(tpls, tpl)
			txTemplates[key] = tpls
		}
		builders, ok := txBuilders[key]
		if !ok {
			txBuilders[key] = []*txbuilder.TemplateBuilder{builder}
		} else {
			builders = append(builders, builder)
			txBuilders[key] = builders
		}
		err = mergeSpendActionUTXO(act, tpl, txTemplates, txBuilders, maxTime, timeRange)
		if err != nil {
			return err
		}
	}

	return nil
}

func TxOutToUtxos(tx *types.Tx, statusFail bool, vaildHeight uint64) []*UTXO {
	utxos := []*UTXO{}
	if tx == nil {
		return utxos
	}
	for i, out := range tx.Outputs {
		bcOut, err := tx.Output(*tx.ResultIds[i])
		if err != nil {
			continue
		}

		if statusFail && *out.AssetAmount.AssetId != *consensus.BTMAssetID {
			continue
		}

		utxos = append(utxos, &UTXO{
			OutputID:       *tx.OutputID(i),
			AssetID:        *out.AssetAmount.AssetId,
			Amount:         out.Amount,
			ControlProgram: out.ControlProgram,
			SourceID:       *bcOut.Source.Ref,
			SourcePos:      bcOut.Source.Position,
			ValidHeight:    vaildHeight,
		})
	}
	return utxos
}

func actTemplatesKey(accID string, assetId *bc.AssetID) string {
	key := accID + assetId.String()
	return key
}

// MergeUTXO
func MergeSpendActionUTXO(ctx context.Context, actions []txbuilder.Action, maxTime time.Time, timeRange uint64) (map[string][]*txbuilder.Template, error) {
	actionTxTemplates := make(map[string][]*txbuilder.Template)
	actionTxBuilder := make(map[string][]*txbuilder.TemplateBuilder)

	for _, act := range actions {
		switch act := act.(type) {
		case *spendAction:
			preTxTemplate := new(txbuilder.Template)
			err := mergeSpendActionUTXO(act, preTxTemplate, actionTxTemplates, actionTxBuilder, maxTime, timeRange)
			if err != nil {
				for _, builders := range actionTxBuilder {
					for _, build := range builders {
						build.Rollback()
					}
				}
				return nil, err
			}
			// rollback reserved utxo
		default:
		}
	}
	return actionTxTemplates, nil
}

// MergeSpendAction merge common assetID and accountID spend action
func GetUTXO(act *spendAction, preTemplate *txbuilder.Template) ([]*UTXO, error) {
	resultUtxos := []*UTXO{}
	validPreTxUTXO := []*UTXO{}
	preTxUTXO := TxOutToUtxos(preTemplate.Transaction, false, 0)
	preTxUTXOAmount := uint64(0)
	for _, v := range preTxUTXO {
		if v.AssetID == *act.AssetId {
			preTxUTXOAmount += v.Amount
			validPreTxUTXO = append(validPreTxUTXO, v)
		}
	}
	utxos, immatureAmount := act.accounts.utxoKeeper.findUtxos(act.AccountID, act.AssetId, true)
	optUtxos, optAmount, reservedAmount := act.accounts.utxoKeeper.optUTXOs(utxos, act.Amount-preTxUTXOAmount)
	if optAmount+reservedAmount+immatureAmount+preTxUTXOAmount < act.Amount {
		return nil, ErrInsufficient
	}

	if optAmount+reservedAmount+preTxUTXOAmount < act.Amount {
		return nil, ErrImmature
	}

	if optAmount+preTxUTXOAmount < act.Amount {
		return nil, ErrReserved
	}
	resultUtxos = append(resultUtxos, validPreTxUTXO[:]...)
	resultUtxos = append(resultUtxos, optUtxos[:]...)

	return resultUtxos, nil
}

func (a *spendAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder, txTemplates map[string][]*txbuilder.Template) error {
	var missing []string
	if a.AccountID == "" {
		missing = append(missing, "account_id")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	acct, err := a.accounts.FindByID(a.AccountID)
	if err != nil {
		return errors.Wrap(err, "get account info")
	}
	preTxUTXOAmount := uint64(0)
	validPreTxUTXOs := []*UTXO{}

	if txTemplates != nil {
		key := actTemplatesKey(a.AccountID, a.AssetId)
		tpls, ok := txTemplates[key]
		if ok {
			preTx := tpls[len(tpls)-1].Transaction
			preTxUTXOs := TxOutToUtxos(preTx, false, 0)
			for _, utxo := range preTxUTXOs {
				if utxo.AssetID == *a.AssetId {
					preTxUTXOAmount += utxo.Amount
					validPreTxUTXOs = append(validPreTxUTXOs, utxo)
				}
			}
		}
	}

	res, err := a.accounts.utxoKeeper.Reserve(a.AccountID, a.AssetId, a.Amount-preTxUTXOAmount, a.UseUnconfirmed, b.MaxTime())
	if err != nil {
		return errors.Wrap(err, "reserving utxos")
	}

	// Cancel the reservation if the build gets rolled back.
	b.OnRollback(func() { a.accounts.utxoKeeper.Cancel(res.id) })

	for _, r := range validPreTxUTXOs {
		cp, err := a.accounts.GetLocalCtrlProgramByProgram(r.ControlProgram)
		if err != nil {
			return errors.Wrap(err, "get local ctrlProgram by address")
		}
		r.ControlProgramIndex = cp.KeyIndex
		r.Address = cp.Address
		txInput, sigInst, err := UtxoToInputs(acct.Signer, r)
		if err != nil {
			return errors.Wrap(err, "creating inputs")
		}

		if err = b.AddInput(txInput, sigInst); err != nil {
			return errors.Wrap(err, "adding inputs")
		}
	}

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

func (a *spendUTXOAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder, preTxTemplate map[string][]*txbuilder.Template) error {
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

	path := signers.Path(signer, signers.AccountKeySpace, u.ControlProgramIndex)
	if u.Address == "" {
		sigInst.AddWitnessKeys(signer.XPubs, path, signer.Quorum)
		return txInput, sigInst, nil
	}

	address, err := common.DecodeAddress(u.Address, &consensus.ActiveNetParams)
	if err != nil {
		return nil, nil, err
	}

	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		sigInst.AddRawWitnessKeys(signer.XPubs, path, signer.Quorum)
		derivedXPubs := chainkd.DeriveXPubs(signer.XPubs, path)
		derivedPK := derivedXPubs[0].PublicKey()
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness([]byte(derivedPK)))

	case *common.AddressWitnessScriptHash:
		sigInst.AddRawWitnessKeys(signer.XPubs, path, signer.Quorum)
		path := signers.Path(signer, signers.AccountKeySpace, u.ControlProgramIndex)
		derivedXPubs := chainkd.DeriveXPubs(signer.XPubs, path)
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
		return m.insertControlPrograms(acps...)
	})
}
