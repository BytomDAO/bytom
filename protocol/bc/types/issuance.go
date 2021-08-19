package types

import (
	"io"

	"github.com/bytom/bytom/crypto/sha3pool"
	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
)

var errBadAssetID = errors.New("asset ID does not match other issuance parameters")

// IssuanceInput satisfies the TypedInput interface and represents a issuance.
type IssuanceInput struct {
	Nonce  []byte
	Amount uint64

	AssetDefinition []byte
	VMVersion       uint64
	IssuanceProgram []byte
	Arguments       [][]byte

	assetId bc.AssetID
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

// AssetDefinitionHash return the hash of the issuance asset definition.
func (ii *IssuanceInput) AssetDefinitionHash() (defhash bc.Hash) {
	sha := sha3pool.Get256()
	defer sha3pool.Put256(sha)
	sha.Write(ii.AssetDefinition)
	defhash.ReadFrom(sha)
	return defhash
}

// AssetID calculate the assetID of the issuance input.
func (ii *IssuanceInput) AssetID() bc.AssetID {
	if ii.assetId.IsZero() {
		ii.assetId = ii.calcAssetID()
	}

	return ii.assetId
}

// InputType is the interface function for return the input type.
func (ii *IssuanceInput) InputType() uint8 { return IssuanceInputType }

// NonceHash return the hash of the issuance asset definition.
func (ii *IssuanceInput) NonceHash() (hash bc.Hash) {
	sha := sha3pool.Get256()
	defer sha3pool.Put256(sha)
	sha.Write(ii.Nonce)
	hash.ReadFrom(sha)
	return hash
}

func (ii *IssuanceInput) calcAssetID() bc.AssetID {
	defhash := ii.AssetDefinitionHash()
	return bc.ComputeAssetID(ii.IssuanceProgram, ii.VMVersion, &defhash)
}

func (ii *IssuanceInput) readCommitment(r *blockchain.Reader) (err error) {
	if ii.Nonce, err = blockchain.ReadVarstr31(r); err != nil {
		return
	}

	if _, err = ii.assetId.ReadFrom(r); err != nil {
		return
	}

	ii.Amount, err = blockchain.ReadVarint63(r)
	return
}

func (ii *IssuanceInput) readWitness(r *blockchain.Reader) (err error) {
	if ii.AssetDefinition, err = blockchain.ReadVarstr31(r); err != nil {
		return err
	}

	if ii.VMVersion, err = blockchain.ReadVarint63(r); err != nil {
		return err
	}

	if ii.IssuanceProgram, err = blockchain.ReadVarstr31(r); err != nil {
		return err
	}

	if ii.calcAssetID() != ii.assetId {
		return errBadAssetID
	}

	if ii.Arguments, err = blockchain.ReadVarstrList(r); err != nil {
		return err
	}

	return nil
}

func (ii *IssuanceInput) writeCommitment(w io.Writer, _ uint64) error {
	if _, err := w.Write([]byte{IssuanceInputType}); err != nil {
		return err
	}

	if _, err := blockchain.WriteVarstr31(w, ii.Nonce); err != nil {
		return err
	}

	assetID := ii.AssetID()
	if _, err := assetID.WriteTo(w); err != nil {
		return err
	}

	_, err := blockchain.WriteVarint63(w, ii.Amount)
	return err
}

func (ii *IssuanceInput) writeWitness(w io.Writer) error {
	if _, err := blockchain.WriteVarstr31(w, ii.AssetDefinition); err != nil {
		return err
	}

	if _, err := blockchain.WriteVarint63(w, ii.VMVersion); err != nil {
		return err
	}

	if _, err := blockchain.WriteVarstr31(w, ii.IssuanceProgram); err != nil {
		return err
	}

	_, err := blockchain.WriteVarstrList(w, ii.Arguments)
	return err
}
