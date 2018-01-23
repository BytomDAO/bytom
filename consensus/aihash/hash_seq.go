package aihash

import (
	"github.com/bytom/protocol/bc"
)

var Hash128 = make(chan [128]bc.Hash)

func Observer() [128]bc.Hash {
	hash128 := <-Hash128

	return hash128
}

func Notify(hash [128]bc.Hash) {
	Hash128 <- hash
}
