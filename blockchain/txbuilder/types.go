package txbuilder

import (
	"context"

	"github.com/bytom/crypto/ed25519/chainkd"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

// Template represents a partially- or fully-signed transaction.
type Template struct {
	Transaction         *legacy.Tx            `json:"raw_transaction"`
	SigningInstructions []*SigningInstruction `json:"signing_instructions"`

	// Local indicates that all inputs to the transaction are signed
	// exclusively by keys managed by this Core. Whenever accepting
	// a template from an external Core, `Local` should be set to
	// false.
	Local bool `json:"local"`

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
}

// Receiver encapsulates information about where to send assets.
type Receiver struct {
	ControlProgram chainjson.HexBytes `json:"control_program,omitempty"`
	Address        string             `json:"address,omitempty"`
}

// AccountPubkey is structure of account pubkey
type AccountPubkey struct {
	Root   chainkd.XPub `json:"root_xpub"`
	Pubkey string       `json:"pubkey"`
	Path   []string     `json:"pubkey_derivation_path"`
}
