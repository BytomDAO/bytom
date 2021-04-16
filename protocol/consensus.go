package protocol

import (
	"github.com/bytom/bytom/crypto/ed25519"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
)

var (
	// ErrDoubleSignBlock represent the consensus is double sign in same height of different block
	ErrDoubleSignBlock  = errors.New("the consensus is double sign in same height of different block")
	errInvalidSignature = errors.New("the signature of block is invalid")
)

var (
	SignatureLength = ed25519.SignatureSize
)

// ICasper interface of casper consensus
type ICasper interface {
	BestChain() (uint64, bc.Hash)
}

func (c *Chain) Casper() ICasper {
	return c.casper
}
