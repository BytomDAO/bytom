package aihash

import (
	"github.com/bytom/consensus/aihash/matrix"
	"github.com/bytom/protocol/bc"
)

// CreateSeed return epoch seed, type is *bc.Hash
func CreateSeed(preSeed *bc.Hash, preEpochBlockHash []*bc.Hash) *bc.Hash {
	return bytesToPointerHash(createSeed(preSeed, preEpochBlockHash))
}

// CreateCache return cache, type is []int32
func CreateCache(seed *bc.Hash) []uint32 {
	cache := make([]uint32, cacheLength/4)
	generateCache(cache, (*seed).Bytes())

	return cache
}

// AIHash verify header is correct.
func AIHash(cache []uint32, height uint64, header *bc.Hash) *bc.Hash {
	matList := make([]matrix.Matrix, matNum)
	fillMatrixList(matList, cache, height)
	m := mulMatrix(matList, (*header).Bytes())
	h := hashMatrix(m)

	return h
}
