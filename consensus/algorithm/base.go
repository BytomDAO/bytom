package algorithm

import "github.com/bytom/protocol/bc"

func CreateCache(seed *bc.Hash) ([]uint32, error) {
	return []uint32{}, nil
}

func CreateSeed(height uint64, preSeed *bc.Hash, preBlockHashs []*bc.Hash) *bc.Hash {
	return preBlockHashs[0]
}

func AIHash(height uint64, blockHash *bc.Hash, seedCache []uint32) (*bc.Hash, error) {
	return blockHash, nil
}
