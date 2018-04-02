package tensority

import "github.com/bytom/protocol/bc"

// Leave here for implement cache
type AIHash struct{}

func Hash(hash, seed *bc.Hash) *bc.Hash {
	cache := calcSeedCache(seed.Bytes())
	data := mulMatrix(hash.Bytes(), cache)

	return hashMatrix(data)
}
