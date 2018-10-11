package types

import (
	"github.com/bytom/crypto/sm3"
	"github.com/bytom/protocol/bc"
)

// IssuanceInput satisfies the TypedInput interface and represents a issuance.
type IssuanceInput struct {
	Nonce  []byte
	Amount uint64

	AssetDefinition []byte
	VMVersion       uint64
	IssuanceProgram []byte
	Arguments       [][]byte
}

// NewIssuanceInput create a new IssuanceInput struct.
func NewIssuanceInput(nonce []byte, amount uint64, issuanceProgram []byte, arguments [][]byte, assetDefinition []byte) *TxInput {
	return &TxInput{
		AssetVersion: 1,
		TypedInput: &IssuanceInput{
			Nonce:           nonce,
			Amount:          amount,
			AssetDefinition: assetDefinition,
			VMVersion:       1,
			IssuanceProgram: issuanceProgram,
			Arguments:       arguments,
		},
	}
}

// InputType is the interface function for return the input type.
func (ii *IssuanceInput) InputType() uint8 { return IssuanceInputType }

// AssetID calculate the assetID of the issuance input.
func (ii *IssuanceInput) AssetID() bc.AssetID {
	defhash := ii.AssetDefinitionHash()
	return bc.ComputeAssetID(ii.IssuanceProgram, ii.VMVersion, &defhash)
}

// AssetDefinitionHash return the hash of the issuance asset definition.
func (ii *IssuanceInput) AssetDefinitionHash() (defhash bc.Hash) {
	hasher := sm3.New()
	hasher.Write(ii.AssetDefinition)
	var b32 [32]byte
	copy(b32[:], hasher.Sum(nil))
	defhash = bc.NewHash(b32)
	return defhash
}

// NonceHash return the hash of the issuance asset definition.
func (ii *IssuanceInput) NonceHash() (hash bc.Hash) {
	h := sm3.New()
	h.Write(ii.Nonce)
	var b32 [32]byte
	copy(b32[:], h.Sum(nil))
	hash = bc.NewHash(b32)
	return hash
}
