package protocol

import (
	"crypto/ed25519"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc/types"
)

var (
	// ErrDoubleSignBlock represent the consensus is double sign in same height of different block
	ErrDoubleSignBlock  = errors.New("the consensus is double sign in same height of different block")
	errInvalidSignature = errors.New("the signature of block is invalid")
)

var (
	SignatureLength = ed25519.SignatureSize
)

func (c *Chain) SignBlockHeader(blockHeader *types.BlockHeader) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	xprv := config.CommonConfig.PrivateKey()
	signature := xprv.Sign(blockHeader.Hash().Bytes())
	blockHeader.Set(signature)
}
