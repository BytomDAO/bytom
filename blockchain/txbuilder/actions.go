package txbuilder

import (
	"context"
	stdjson "encoding/json"
	"errors"

	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/encoding/json"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/protocol/vm/vmutil"
)

var retirementProgram = []byte{byte(vm.OP_FAIL)}

// DecodeControlReceiverAction convert input data to action struct
func DecodeControlReceiverAction(data []byte) (Action, error) {
	a := new(controlReceiverAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type controlReceiverAction struct {
	bc.AssetAmount
	Receiver      *Receiver `json:"receiver"`
	ReferenceData json.Map  `json:"reference_data"`
}

func (a *controlReceiverAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.Receiver == nil {
		missing = append(missing, "receiver")
	} else {
		if len(a.Receiver.ControlProgram) == 0 {
			missing = append(missing, "receiver.control_program")
		}
		if a.Receiver.ExpiresAt.IsZero() {
			missing = append(missing, "receiver.expires_at")
		}
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	b.RestrictMaxTime(a.Receiver.ExpiresAt)
	out := legacy.NewTxOutput(*a.AssetId, a.Amount, a.Receiver.ControlProgram, a.ReferenceData)
	return b.AddOutput(out)
}

// DecodeControlAddressAction convert input data to action struct
func DecodeControlAddressAction(data []byte) (Action, error) {
	a := new(controlAddressAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type controlAddressAction struct {
	bc.AssetAmount
	Address       string   `json:"address"`
	ReferenceData json.Map `json:"reference_data"`
}

func (a *controlAddressAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.Address == "" {
		missing = append(missing, "address")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	address, err := common.DecodeAddress(a.Address, &consensus.MainNetParams)
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

	out := legacy.NewTxOutput(*a.AssetId, a.Amount, program, a.ReferenceData)
	return b.AddOutput(out)
}

// DecodeControlProgramAction convert input data to action struct
func DecodeControlProgramAction(data []byte) (Action, error) {
	a := new(controlProgramAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type controlProgramAction struct {
	bc.AssetAmount
	Program       json.HexBytes `json:"control_program"`
	ReferenceData json.Map      `json:"reference_data"`
}

func (a *controlProgramAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if len(a.Program) == 0 {
		missing = append(missing, "control_program")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	out := legacy.NewTxOutput(*a.AssetId, a.Amount, a.Program, a.ReferenceData)
	return b.AddOutput(out)
}

// DecodeSetTxRefDataAction convert input data to action struct
func DecodeSetTxRefDataAction(data []byte) (Action, error) {
	a := new(setTxRefDataAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type setTxRefDataAction struct {
	Data json.Map `json:"reference_data"`
}

func (a *setTxRefDataAction) Build(ctx context.Context, b *TemplateBuilder) error {
	if len(a.Data) == 0 {
		return MissingFieldsError("reference_data")
	}
	return b.setReferenceData(a.Data)
}

// DecodeRetireAction convert input data to action struct
func DecodeRetireAction(data []byte) (Action, error) {
	a := new(retireAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type retireAction struct {
	bc.AssetAmount
	ReferenceData json.Map `json:"reference_data"`
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

	out := legacy.NewTxOutput(*a.AssetId, a.Amount, retirementProgram, a.ReferenceData)
	return b.AddOutput(out)
}
