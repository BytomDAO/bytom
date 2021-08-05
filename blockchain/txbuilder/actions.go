package txbuilder

import (
	"context"
	stdjson "encoding/json"
	"errors"

	"github.com/bytom/bytom/common"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/encoding/json"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/vm/vmutil"
)

// DecodeControlAddressAction convert input data to action struct
func DecodeControlAddressAction(data []byte) (Action, error) {
	a := new(controlAddressAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type controlAddressAction struct {
	bc.AssetAmount
	Address string `json:"address"`
}

func (a *controlAddressAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.Address == "" {
		missing = append(missing, "address")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	address, err := common.DecodeAddress(a.Address, &consensus.ActiveNetParams)
	if err != nil {
		return err
	}
	redeemContract := address.ScriptAddress()
	program := []byte{}

	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return errors.New("unsupport address type")
	}
	if err != nil {
		return err
	}

	out := types.NewOriginalTxOutput(*a.AssetId, a.Amount, program, nil)
	return b.AddOutput(out)
}

func (a *controlAddressAction) ActionType() string {
	return "control_address"
}

// DecodeControlProgramAction convert input data to action struct
func DecodeControlProgramAction(data []byte) (Action, error) {
	a := new(controlProgramAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type controlProgramAction struct {
	bc.AssetAmount
	Program json.HexBytes `json:"control_program"`
}

func (a *controlProgramAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if len(a.Program) == 0 {
		missing = append(missing, "control_program")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	out := types.NewOriginalTxOutput(*a.AssetId, a.Amount, a.Program, nil)
	return b.AddOutput(out)
}

func (a *controlProgramAction) ActionType() string {
	return "control_program"
}

// DecodeRetireAction convert input data to action struct
func DecodeRetireAction(data []byte) (Action, error) {
	a := new(retireAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type retireAction struct {
	bc.AssetAmount
	Arbitrary json.HexBytes `json:"arbitrary"`
}

func (a *retireAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	program, err := vmutil.RetireProgram(a.Arbitrary)
	if err != nil {
		return err
	}
	out := types.NewOriginalTxOutput(*a.AssetId, a.Amount, program, nil)
	return b.AddOutput(out)
}

func (a *retireAction) ActionType() string {
	return "retire"
}

// DecodeRegisterAction convert input data to action struct
func DecodeRegisterAction(data []byte) (Action, error) {
	a := new(registerAction)
	return a, stdjson.Unmarshal(data, a)
}

type registerAction struct {
	bc.AssetAmount
	Contract json.HexBytes `json:"contract"`
}

func (a *registerAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(a.Contract) == 0 {
		missing = append(missing, "contract")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	if a.AssetId.String() != consensus.BTMAssetID.String() {
		return errors.New("register contract action asset must be BTM")
	}

	if a.Amount < consensus.BCRPRequiredBTMAmount {
		return errors.New("less than BCRP required BTM amount")
	}

	program, err := vmutil.RegisterProgram(a.Contract)
	if err != nil {
		return err
	}
	out := types.NewOriginalTxOutput(*a.AssetId, a.Amount, program, [][]byte{})
	return b.AddOutput(out)
}

func (a *registerAction) ActionType() string {
	return "register_contract"
}

// DecodeVoteOutputAction convert input data to action struct
func DecodeVoteOutputAction(data []byte) (Action, error) {
	a := new(voteOutputAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type voteOutputAction struct {
	bc.AssetAmount
	Address string        `json:"address"`
	Vote    json.HexBytes `json:"vote"`
}

func (a *voteOutputAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.Address == "" {
		missing = append(missing, "address")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(a.Vote) == 0 {
		missing = append(missing, "vote")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	address, err := common.DecodeAddress(a.Address, &consensus.ActiveNetParams)
	if err != nil {
		return err
	}

	redeemContract := address.ScriptAddress()
	program := []byte{}
	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return errors.New("unsupport address type")
	}
	if err != nil {
		return err
	}

	out := types.NewVoteOutput(*a.AssetId, a.Amount, program, a.Vote, nil)
	return b.AddOutput(out)
}

func (a *voteOutputAction) ActionType() string {
	return "vote_output"
}
