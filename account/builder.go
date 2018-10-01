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
			reservedUTXO, err := act.reserveUTXO(maxTime)
			if err != nil {
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

func (a *spendAction) reserveUTXO(maxTime time.Time) (*ActionReservedUTXO, error) {
	resUtxo := newActionReservedUTXO()
	for gasAmount := uint64(0); resUtxo.totalAmount <= gasAmount+a.Amount; gasAmount = calcMergeNum(uint64(len(resUtxo.utxos))) * MergeSpendActionUTXOGas {
		reserveAmount := a.Amount + gasAmount - resUtxo.totalAmount
		res, err := a.accounts.utxoKeeper.Reserve(a.AccountID, a.AssetId, reserveAmount, a.UseUnconfirmed, maxTime)
		if err != nil {
			for _, resID := range resUtxo.IDs {
				a.accounts.utxoKeeper.Cancel(resID)
			}
			return nil, err
		}

		resUtxo.IDs = append(resUtxo.IDs, res.id)
		resUtxo.totalAmount += reserveAmount + res.change
		resUtxo.utxos = append(resUtxo.utxos, res.utxos[:]...)
	}
	return resUtxo, nil
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

	tpls := []*txbuilder.Template{}

	buildAmount := uint64(0)
	builder := &txbuilder.TemplateBuilder{}
	for index := 0; index < len(utxos); index++ {
		input, sigInst, err := UtxoToInputs(acct.Signer, utxos[index])
		if err != nil {
			return nil, nil, err
		}

		if err = builder.AddInput(input, sigInst); err != nil {
			return nil, nil, err
		}

		buildAmount += input.Amount()
		if builder.InputCount() != TxMaxInputUTXONum && index != len(utxos)-1 {
			continue
		}

		outAmount := buildAmount - MergeSpendActionUTXOGas
		output := types.NewTxOutput(*a.AssetId, outAmount, acp.ControlProgram)
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
			AssetID:             *a.AssetId,
			Amount:              outAmount,
			ControlProgram:      acp.ControlProgram,
			SourceID:            *bcOut.Source.Ref,
			SourcePos:           bcOut.Source.Position,
			ControlProgramIndex: acp.KeyIndex,
			Address:             acp.Address,
		})

		tpls = append(tpls, tpl)
		if outAmount >= a.Amount {
			break
		}

		buildAmount = 0
		builder = &txbuilder.TemplateBuilder{}
	}
	return tpls, utxos[len(utxos)-1], nil
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
