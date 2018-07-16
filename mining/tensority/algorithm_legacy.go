// +build !cgo

package tensority

import (
    "github.com/bytom/protocol/bc"
)

func algorithm(bh, seed *bc.Hash) *bc.Hash {
    cache := calcSeedCache(seed.Bytes())
    data := mulMatrix(bh.Bytes(), cache)
    return hashMatrix(data)
}
