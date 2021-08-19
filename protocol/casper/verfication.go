package casper

import (
	"bytes"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/sha3"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

var errVerifySignature = errors.New("signature of verification message is invalid")

type ValidCasperSignMsg struct {
	SourceHash bc.Hash
	TargetHash bc.Hash
	Signature  []byte
	PubKey     string
}

// verification represent a verification message for the block
// source hash and target hash point to the checkpoint, and the source checkpoint is the target checkpoint's parent(not be directly)
// the vector <sourceHash, targetHash, sourceHeight, targetHeight, pubKey> as the message of signature
type verification struct {
	SourceHash   bc.Hash
	TargetHash   bc.Hash
	SourceHeight uint64
	TargetHeight uint64
	Signature    []byte
	PubKey       string
	order        int
}

func convertVerification(source, target *state.Checkpoint, msg *ValidCasperSignMsg) (*verification, error) {
	validators := target.Parent.EffectiveValidators()
	if _, ok := validators[msg.PubKey]; !ok {
		return nil, errPubKeyIsNotValidator
	}

	return &verification{
		SourceHash:   source.Hash,
		TargetHash:   target.Hash,
		SourceHeight: source.Height,
		TargetHeight: target.Height,
		Signature:    msg.Signature,
		PubKey:       msg.PubKey,
		order:        validators[msg.PubKey].Order,
	}, nil
}

func supLinkToVerifications(source, target *state.Checkpoint, supLink *types.SupLink) []*verification {
	var result []*verification
	for _, validator := range target.Parent.EffectiveValidators() {
		if signature := supLink.Signatures[validator.Order]; len(signature) != 0 {
			result = append(result, &verification{
				SourceHash:   source.Hash,
				TargetHash:   target.Hash,
				SourceHeight: source.Height,
				TargetHeight: target.Height,
				Signature:    signature,
				PubKey:       validator.PubKey,
				order:        validator.Order,
			})
		}
	}
	return result
}

// Sign used to sign the verification by specified xPrv
func (v *verification) Sign(xPrv chainkd.XPrv) error {
	message, err := v.encodeMessage()
	if err != nil {
		return err
	}

	v.Signature = xPrv.Sign(message)
	return nil
}

func (v *verification) toValidCasperSignMsg() ValidCasperSignMsg {
	return ValidCasperSignMsg{
		SourceHash: v.SourceHash,
		TargetHash: v.TargetHash,
		Signature:  v.Signature,
		PubKey:     v.PubKey,
	}
}

func (v *verification) valid() error {
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
func (v *verification) verifySignature() error {
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
func (v *verification) encodeMessage() ([]byte, error) {
	buff := new(bytes.Buffer)
	if _, err := v.SourceHash.WriteTo(buff); err != nil {
		return nil, err
	}

	if _, err := v.TargetHash.WriteTo(buff); err != nil {
		return nil, err
	}

	msg := sha3.Sum256(buff.Bytes())
	return msg[:], nil
}
