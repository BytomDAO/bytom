package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"

	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/crypto/sha3pool"
	"github.com/bytom/bytom/protocol/bc"
)

var errVerifySignature = errors.New("signature of verification message is invalid")

// Verification represent a verification message for the block
// source hash and target hash point to the checkpoint, and the source checkpoint is the target checkpoint's parent(not be directly)
// the vector <sourceHash, targetHash, sourceHeight, targetHeight, pubKey> as the message of signature
type Verification struct {
	SourceHash   bc.Hash
	TargetHash   bc.Hash
	SourceHeight uint64
	TargetHeight uint64
	Signature    string
	PubKey       string
}

// EncodeMessage encode the verification for the validators to sign or verify
func (v *Verification) EncodeMessage() ([]byte, error) {
	buff := new(bytes.Buffer)
	if _, err := v.SourceHash.WriteTo(buff); err != nil {
		return nil, err
	}

	if _, err := v.TargetHash.WriteTo(buff); err != nil {
		return nil, err
	}

	uint64Buff := make([]byte, 8)

	binary.LittleEndian.PutUint64(uint64Buff, v.SourceHeight)
	if _, err := buff.Write(uint64Buff); err != nil {
		return nil, err
	}

	binary.LittleEndian.PutUint64(uint64Buff, v.TargetHeight)
	if _, err := buff.Write(uint64Buff); err != nil {
		return nil, err
	}

	return sha3Hash(buff.Bytes())
}

// Sign used to sign the verification by specified xPrv
func (v *Verification) Sign(xPrv chainkd.XPrv) error {
	message, err := v.EncodeMessage()
	if err != nil {
		return err
	}

	v.Signature = hex.EncodeToString(xPrv.Sign(message))
	return nil
}

// VerifySignature verify the signature of encode message of verification
func (v *Verification) VerifySignature() error {
	pubKey, err := hex.DecodeString(v.PubKey)
	if err != nil {
		return err
	}

	signature, err := hex.DecodeString(v.Signature)
	if err != nil {
		return err
	}

	message, err := v.EncodeMessage()
	if err != nil {
		return err
	}

	var xPub chainkd.XPub
	copy(xPub[:], pubKey)
	if !xPub.Verify(message, signature) {
		return errVerifySignature
	}

	return nil
}

func sha3Hash(message []byte) ([]byte, error) {
	sha3 := sha3pool.Get256()
	defer sha3pool.Put256(sha3)

	if _, err := sha3.Write(message); err != nil {
		return nil, err
	}

	hash := &bc.Hash{}
	if _, err := hash.ReadFrom(sha3); err != nil {
		return nil, err
	}

	return hash.Bytes(), nil
}
