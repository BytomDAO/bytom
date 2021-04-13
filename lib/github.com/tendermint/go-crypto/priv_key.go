package crypto

import (
	"encoding/hex"

	"github.com/tendermint/ed25519"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/go-wire/data"
)

func PrivKeyFromBytes(privKeyBytes []byte) (privKey PrivKey, err error) {
	err = wire.ReadBinaryBytes(privKeyBytes, &privKey)
	if err == nil {
		// add support for a ValidateKey method on PrivKeys
		// to make sure they load correctly
		val, ok := privKey.Unwrap().(validatable)
		if ok {
			err = val.ValidateKey()
		}
	}
	return
}

// validatable is an optional interface for keys that want to
// check integrity
type validatable interface {
	ValidateKey() error
}

//----------------------------------------

// DO NOT USE THIS INTERFACE.
// You probably want to use PrivKey
// +gen wrapper:"PrivKey,Impl[PrivKeyEd25519,PrivKeySecp256k1],ed25519,secp256k1"
type PrivKeyInner interface {
	Sign(msg []byte) Signature
	PubKey() PubKey
	Wrap() PrivKey
}

//-------------------------------------

var _ PrivKeyInner = PrivKeyEd25519{}

// Implements PrivKey
type PrivKeyEd25519 [64]byte

func (privKey PrivKeyEd25519) Sign(msg []byte) Signature {
	privKeyBytes := [64]byte(privKey)
	signatureBytes := ed25519.Sign(&privKeyBytes, msg)
	return SignatureEd25519(*signatureBytes).Wrap()
}

func (privKey PrivKeyEd25519) PubKey() PubKey {
	privKeyBytes := [64]byte(privKey)
	pubBytes := *ed25519.MakePublicKey(&privKeyBytes)
	return PubKeyEd25519(pubBytes).Wrap()
}

func (p PrivKeyEd25519) MarshalJSON() ([]byte, error) {
	return data.Encoder.Marshal(p[:])
}

func (p *PrivKeyEd25519) UnmarshalJSON(enc []byte) error {
	var ref []byte
	err := data.Encoder.Unmarshal(&ref, enc)
	copy(p[:], ref)
	return err
}

func (privKey PrivKeyEd25519) String() string {
	return hex.EncodeToString(privKey[:])
}

func GenPrivKeyEd25519() PrivKeyEd25519 {
	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], CRandBytes(32))
	ed25519.MakePublicKey(privKeyBytes)
	return PrivKeyEd25519(*privKeyBytes)
}
