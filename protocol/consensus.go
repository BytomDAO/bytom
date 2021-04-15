package protocol

import (
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/consensus"
)

var (
	// ErrDoubleSignBlock represent the consensus is double sign in same height of different block
	ErrDoubleSignBlock  = errors.New("the consensus is double sign in same height of different block")
	errInvalidSignature = errors.New("the signature of block is invalid")
)

func (c *Chain) Casper() *consensus.Casper {
	return c.casper
}
