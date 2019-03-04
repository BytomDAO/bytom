package crypto

import (
	"github.com/tendermint/go-wire"
	data "github.com/tendermint/go-wire/data"
)

func SignatureFromBytes(sigBytes []byte) (sig Signature, err error) {
	err = wire.ReadBinaryBytes(sigBytes, &sig)
	return
}

//----------------------------------------

// DO NOT USE THIS INTERFACE.
// You probably want to use Signature.
// +gen wrapper:"Signature,Impl[SignatureEd25519,SignatureSecp256k1],ed25519,secp256k1"
type SignatureInner interface {
	Bytes() []byte
	Wrap() Signature
}

//-------------------------------------

var _ SignatureInner = SignatureEd25519{}

// Implements Signature
type SignatureEd25519 [64]byte

func (sig SignatureEd25519) Bytes() []byte {
	return wire.BinaryBytes(Signature{sig})
}

func (sig SignatureEd25519) MarshalJSON() ([]byte, error) {
	return data.Encoder.Marshal(sig[:])
}

func (sig *SignatureEd25519) UnmarshalJSON(enc []byte) error {
	var ref []byte
	err := data.Encoder.Unmarshal(&ref, enc)
	copy(sig[:], ref)
	return err
}
