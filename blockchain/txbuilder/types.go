package txbuilder

import (
	"context"
	"encoding/json"

	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	chainjson "github.com/bytom/bytom/encoding/json"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

// Template represents a partially- or fully-signed transaction.
type Template struct {
	Transaction         *types.Tx             `json:"raw_transaction"`
	SigningInstructions []*SigningInstruction `json:"signing_instructions"`
	Fee                 uint64                `json:"fee"`
	// AllowAdditional affects whether Sign commits to the tx sighash or
	// to individual details of the tx so far. When true, signatures
	// commit to tx details, and new details may be added but existing
	// ones cannot be changed. When false, signatures commit to the tx
	// as a whole, and any change to the tx invalidates the signature.
	AllowAdditional bool `json:"allow_additional_actions"`
}

// Hash return sign hash
func (t *Template) Hash(idx uint32) bc.Hash {
	return t.Transaction.SigHash(idx)
}

// Action is a interface
type Action interface {
	Build(context.Context, *TemplateBuilder) error
	ActionType() string
}

// Receiver encapsulates information about where to send assets.
type Receiver struct {
	ControlProgram chainjson.HexBytes `json:"control_program,omitempty"`
	Address        string             `json:"address,omitempty"`
}

// ContractArgument for smart contract
type ContractArgument struct {
	Type    string          `json:"type"`
	RawData json.RawMessage `json:"raw_data"`
}

// RawTxSigArgument is signature-related argument for run contract
type RawTxSigArgument struct {
	RootXPub chainkd.XPub         `json:"xpub"`
	Path     []chainjson.HexBytes `json:"derivation_path"`
}

// DataArgument is the other argument for run contract
type DataArgument struct {
	Value chainjson.HexBytes `json:"value"`
}

// StrArgument is the string argument for run contract
type StrArgument struct {
	Value string `json:"value"`
}

// IntegerArgument is the integer argument for run contract
type IntegerArgument struct {
	Value int64 `json:"value"`
}

// BoolArgument is the boolean argument for run contract
type BoolArgument struct {
	Value bool `json:"value"`
}
