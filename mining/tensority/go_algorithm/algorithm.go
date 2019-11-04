package go_algorithm

import (
	"github.com/bytom/bytom/protocol/bc"
)

func LegacyAlgorithm(bh, seed *bc.Hash) *bc.Hash {
	cache := calcSeedCache(seed.Bytes())
	data := mulMatrix(bh.Bytes(), cache)
	return hashMatrix(data)
}
