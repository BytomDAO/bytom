package protocol

import (
	"crypto/ed25519"

	"github.com/bytom/bytom/errors"
)

var (
	// ErrDoubleSignBlock represent the consensus is double sign in same height of different block
	ErrDoubleSignBlock  = errors.New("the consensus is double sign in same height of different block")
	errInvalidSignature = errors.New("the signature of block is invalid")
)

var (
	SignatureLength = ed25519.SignatureSize
)
