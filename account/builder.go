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

const (
	//TxMaxInputUTXONum maximum utxo quantity in a tx
	TxMaxInputUTXONum = 10
	//MergeSpendActionUTXOGas chain tx gas
	MergeSpendActionUTXOGas = 10000000
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

func getProgramFromAddress(addr string) ([]byte, error) {
	address, err := common.DecodeAddress(addr, &consensus.ActiveNetParams)
	if err != nil {
		return nil, err
	}
	redeemContract := address.ScriptAddress()
	program := []byte{}

	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return nil, errors.New("unsupport address type")
	}
	if err != nil {
		return nil, err
	}
	return program, nil
}

func newTxOutput(assetId *bc.AssetID, amount uint64, address string) *types.TxOutput {
	program, _ := getProgramFromAddress(address)
	out := types.NewTxOutput(*assetId, amount, program)
	return out
}

func txOutToUtxos(tx *types.Tx, cp *CtrlProgram, statusFail bool, vaildHeight uint64) []*UTXO {
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
			OutputID:            *tx.OutputID(i),
			AssetID:             *out.AssetAmount.AssetId,
			Amount:              out.Amount,
			ControlProgram:      out.ControlProgram,
			SourceID:            *bcOut.Source.Ref,
			SourcePos:           bcOut.Source.Position,
			ValidHeight:         vaildHeight,
			Address:             cp.Address,
			ControlProgramIndex: cp.KeyIndex,
		})
	}
	return utxos
}

//calcMergeNum calculate the number of times that n utxos are merged into one
func calcMergeNum(utxoNum uint64) uint64 {
	num := uint64(0)
	for utxoNum != 0 {
		num += utxoNum / TxMaxInputUTXONum
		utxoNum = utxoNum/10 + utxoNum%10
		if utxoNum > 0 && utxoNum < 10 {
			num++
			break
		}
	}
	return num
}

func actTemplatesKey(accID string, assetId *bc.AssetID) string {
	key := accID + assetId.String()
	return key
}

func RollbackResUTXO(utxoKeeper *utxoKeeper, resIDs []uint64) {
	for _, resID := range resIDs {
		utxoKeeper.Cancel(resID)
	}
}

// MergeUTXO
func MergeSpendActionsUTXO(ctx context.Context, actions []txbuilder.Action, maxTime time.Time, timeRange uint64) ([]*txbuilder.Template, []*txbuilder.Action, *MergeActionsUTXOResult, error) {
	actionTxTemplates := make([]*txbuilder.Template, 0)
	otherActions := make([]*txbuilder.Action, 0)
	mergeResult := &MergeActionsUTXOResult{ResIDs: []uint64{}, Outputs: make([]*PreTxOutput, 0)}
	for _, act := range actions {
		switch act := act.(type) {
		case *spendAction:
			reservedUTXO := newActionReservedUTXO()
			if err := act.reserveUTXO(act.Amount, maxTime, reservedUTXO); err != nil {
				return nil, nil, mergeResult, err
			}
			mergeResult.ResIDs = append(mergeResult.ResIDs, reservedUTXO.IDs[:]...)
			tpls, preTxOutput, err := act.mergeSpendActionUTXO(reservedUTXO.utxos, maxTime, timeRange)
			if err != nil {
				return nil, nil, mergeResult, err
			}
			acct, err := act.accounts.FindByID(act.AccountID)
			if err != nil {
				return nil, nil, mergeResult, err
			}
			input, sigInst, err := UtxoToInputs(acct.Signer, preTxOutput)
			if err != nil {
				return nil, nil, mergeResult, err
			}
			output := &PreTxOutput{TxInput: input, Sign: sigInst}
			mergeResult.Outputs = append(mergeResult.Outputs, output)
			actionTxTemplates = append(actionTxTemplates, tpls[:]...)
		default:
			otherActions = append(otherActions, &act)
		}
	}
	return actionTxTemplates, otherActions, mergeResult, nil
}

type ActionReservedUTXO struct {
	IDs         []uint64
	totalAmount uint64
	utxos       []*UTXO
}

func newActionReservedUTXO() *ActionReservedUTXO {
	return &ActionReservedUTXO{
		IDs:   []uint64{},
		utxos: []*UTXO{},
	}
}

type PreTxOutput struct {
	TxInput *types.TxInput
	Sign    *txbuilder.SigningInstruction
}

type MergeActionsUTXOResult struct {
	ResIDs  []uint64
	Outputs []*PreTxOutput
}

func (a *spendAction) reserveUTXO(amount uint64, maxTime time.Time, resUTXO *ActionReservedUTXO) error {
	res, err := a.accounts.utxoKeeper.Reserve(a.AccountID, a.AssetId, amount, a.UseUnconfirmed, maxTime)
	if err != nil {
		//rollback action reserved utxo
		for _, resID := range resUTXO.IDs {
			a.accounts.utxoKeeper.Cancel(resID)
		}
		return err
	}
	resUTXO.IDs = append(resUTXO.IDs, res.id)
	resUTXO.totalAmount += amount + res.change
	resUTXO.utxos = append(resUTXO.utxos, res.utxos[:]...)
	gasRequired := calcMergeNum(uint64(len(resUTXO.utxos))) * MergeSpendActionUTXOGas
	if gasRequired+a.Amount > resUTXO.totalAmount {
		if err := a.reserveUTXO(gasRequired+a.Amount-resUTXO.totalAmount, maxTime, resUTXO); err != nil {
			return err
		}
	}
	return nil
}

func (a *spendAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
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

//mergeSpendActionUTXO combine the n utxos required by SpendAction into 1
func (a *spendAction) mergeSpendActionUTXO(utxos []*UTXO, maxTime time.Time, timeRange uint64) ([]*txbuilder.Template, *UTXO, error) {
	if len(utxos) == 0 {
		return nil, nil, errors.New("mergeSpendActionUTXO utxos num 0")
	}
	acct, err := a.accounts.FindByID(a.AccountID)
	if err != nil {
		return nil, nil, err
	}
	acp, err := a.accounts.GetLocalCtrlProgramByAddress(utxos[0].Address)
	if err != nil {
		return nil, nil, err
	}
	mergeNum := calcMergeNum(uint64(len(utxos)))
	builders := make([]txbuilder.TemplateBuilder, mergeNum)
	tpls := make([]*txbuilder.Template, 0)
	assetAmount := uint64(0)
	for index := 0; index < len(utxos); index++ {
		if index != 0 && index%TxMaxInputUTXONum == 0 {
			builderIndix := uint64(index/TxMaxInputUTXONum) - 1
			output := newTxOutput(a.AssetId, assetAmount-MergeSpendActionUTXOGas, acp.Address)
			if err := builders[builderIndix].AddOutput(output); err != nil {
				return nil, nil, err
			}
			tpl, _, err := builders[builderIndix].Build()
			if err != nil {
				return nil, nil, err
			}
			tpls = append(tpls, tpl)
			preTxOutputs := txOutToUtxos(tpl.Transaction, acp, false, 0)
			utxos = append(utxos, preTxOutputs[:]...)
			assetAmount = 0
		}
		input, sigInst, err := UtxoToInputs(acct.Signer, utxos[index])
		if err != nil {
			return nil, nil, err
		}
		if err = builders[index/TxMaxInputUTXONum].AddInput(input, sigInst); err != nil {
			return nil, nil, err
		}
		assetAmount += input.Amount()
		if index == len(utxos)-1 {
			builderIndix := mergeNum - 1
			output := newTxOutput(a.AssetId, a.Amount, acp.Address)
			if err := builders[builderIndix].AddOutput(output); err != nil {
				return nil, nil, err
			}
			if assetAmount < MergeSpendActionUTXOGas+a.Amount {
				return nil, nil, errors.New("mergeSpendActionUTXO amount err")
			}
			if change := assetAmount - MergeSpendActionUTXOGas - a.Amount; change > 0 {
				changeOutput := newTxOutput(a.AssetId, change, acp.Address)
				builders[builderIndix].AddOutput(changeOutput)
			}
			tpl, _, err := builders[builderIndix].Build()
			if err != nil {
				return nil, nil, err
			}
			tpls = append(tpls, tpl)
			preTxOutputs := txOutToUtxos(tpl.Transaction, acp, false, 0)
			return tpls, preTxOutputs[0], nil
		}
	}
	return nil, nil, errors.New("mergeSpendActionUTXO err")
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
