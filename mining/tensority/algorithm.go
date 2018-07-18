package tensority

import (
    "github.com/bytom/protocol/bc"
)

var (
    UseSIMD = false
)

func legacyAlgorithm(bh, seed *bc.Hash) *bc.Hash {
    cache := calcSeedCache(seed.Bytes())
    data := mulMatrix(bh.Bytes(), cache)
    return hashMatrix(data)
}

func algorithm(bh, seed *bc.Hash) *bc.Hash {
    if UseSIMD {
        return simdAlgorithm(bh, seed)
    } else {
        return legacyAlgorithm(bh, seed)
    }
}
