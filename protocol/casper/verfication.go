package casper

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/sha3"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/protocol/bc"
)

var errVerifySignature = errors.New("signature of verification message is invalid")

type ValidCasperSignEvent struct {
	*Verification
}

// Verification represent a verification message for the block
// source hash and target hash point to the checkpoint, and the source checkpoint is the target checkpoint's parent(not be directly)
// the vector <sourceHash, targetHash, sourceHeight, targetHeight, pubKey> as the message of signature
type Verification struct {
	SourceHash   bc.Hash
	TargetHash   bc.Hash
	SourceHeight uint64
	TargetHeight uint64
	Signature    []byte
	PubKey       string
}

// Sign used to sign the verification by specified xPrv
func (v *Verification) Sign(xPrv chainkd.XPrv) error {
	message, err := v.encodeMessage()
	if err != nil {
		return err
	}

	v.Signature = xPrv.Sign(message)
	return nil
}

func (v *Verification) vaild() error {
	blocksOfEpoch := consensus.ActiveNetParams.BlocksOfEpoch
	if v.SourceHeight%blocksOfEpoch != 0 || v.TargetHeight%blocksOfEpoch != 0 {
		return errVoteToGrowingCheckpoint
	}

	if v.SourceHeight >= v.TargetHeight {
		return errVoteToSameCheckpoint
	}

	return v.verifySignature()
}

// verifySignature verify the signature of encode message of verification
func (v *Verification) verifySignature() error {
	message, err := v.encodeMessage()
	if err != nil {
		return err
	}

	pubKey, err := hex.DecodeString(v.PubKey)
	if err != nil {
		return err
	}

	var xPub chainkd.XPub
	copy(xPub[:], pubKey)
	if !xPub.Verify(message, v.Signature) {
		return errVerifySignature
	}

	return nil
}

// encodeMessage encode the verification for the validators to sign or verify
func (v *Verification) encodeMessage() ([]byte, error) {
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

	msg := sha3.Sum256(buff.Bytes())
	return msg[:], nil
}
