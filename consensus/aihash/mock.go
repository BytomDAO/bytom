package aihash

import (
	"github.com/bytom/protocol/bc"
)

func MockCreateSeed(height uint64, preSeed *bc.Hash, preBlockHashs []*bc.Hash) *bc.Hash {
	return preBlockHashs[0]
}

func MockCreateCache(seed *bc.Hash) ([]uint32, error) {
	return []uint32{}, nil
}

func MockAIHash(height uint64, blockHash *bc.Hash, seedCache []uint32) (*bc.Hash, error) {
	return blockHash, nil
}
