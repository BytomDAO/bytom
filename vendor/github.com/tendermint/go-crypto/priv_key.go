package crypto

import (
	"bytes"

	"github.com/tendermint/ed25519"
	"github.com/tendermint/go-wire"
	data "github.com/tendermint/go-wire/data"
	. "github.com/tendermint/tmlibs/common"
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
	Bytes() []byte
	Sign(msg []byte) Signature
	PubKey() PubKey
	Equals(PrivKey) bool
	Wrap() PrivKey
}

//-------------------------------------

var _ PrivKeyInner = PrivKeyEd25519{}

// Implements PrivKey
type PrivKeyEd25519 [64]byte

func (privKey PrivKeyEd25519) Bytes() []byte {
	return wire.BinaryBytes(PrivKey{privKey})
}

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

func (privKey PrivKeyEd25519) Equals(other PrivKey) bool {
	if otherEd, ok := other.Unwrap().(PrivKeyEd25519); ok {
		return bytes.Equal(privKey[:], otherEd[:])
	} else {
		return false
	}
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
	return Fmt("PrivKeyEd25519{*****}")
}

// Deterministically generates new priv-key bytes from key.
func (privKey PrivKeyEd25519) Generate(index int) PrivKeyEd25519 {
	newBytes := wire.BinarySha256(struct {
		PrivKey [64]byte
		Index   int
	}{privKey, index})
	var newKey [64]byte
	copy(newKey[:], newBytes)
	return PrivKeyEd25519(newKey)
}

func GenPrivKeyEd25519() PrivKeyEd25519 {
	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], CRandBytes(32))
	ed25519.MakePublicKey(privKeyBytes)
	return PrivKeyEd25519(*privKeyBytes)
}
