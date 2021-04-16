package protocol

import (
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/crypto/ed25519"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
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
	Validators(blockHash *bc.Hash) ([]*state.Validator, error)
}

func (c *Chain) Casper() ICasper {
	return c.casper
}

func (c *Chain) GetBlocker(prevBlockHash *bc.Hash, timestamp uint64) (string, error) {
	validators, err := c.casper.Validators(prevBlockHash)
	if err != nil {
		return "", err
	}
	prevVoteRoundLastBlock, err := c.getPrevRoundLastBlock(prevBlockHash)
	if err != nil {
		return "", err
	}

	startTimestamp := prevVoteRoundLastBlock.Timestamp + consensus.ActiveNetParams.BlockTimeInterval
	order := getBlockerOrder(startTimestamp, timestamp, uint64(len(validators)))
	return validators[order].PubKey, nil
}

func getBlockerOrder(startTimestamp, blockTimestamp, numOfConsensusNode uint64) uint64 {
	roundBlockTime := consensus.ActiveNetParams.BlockNumEachNode * numOfConsensusNode * consensus.ActiveNetParams.BlockTimeInterval
	lastRoundStartTime := startTimestamp + (blockTimestamp-startTimestamp)/roundBlockTime*roundBlockTime
	return (blockTimestamp - lastRoundStartTime) / (consensus.ActiveNetParams.BlockNumEachNode * consensus.ActiveNetParams.BlockTimeInterval)
}

func (c *Chain) getPrevRoundLastBlock(hash *bc.Hash) (*types.BlockHeader, error) {
	header, err := c.store.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	// loop find the previous epoch block hash
	for header.Height%state.BlocksOfEpoch != 0 {
		header, err = c.store.GetBlockHeader(&header.PreviousBlockHash)
		if err != nil {
			return nil, err
		}
	}
	return header, nil
}
