package tensority

import (
	"github.com/bytom/crypto"
	"github.com/bytom/crypto/scrypt"
	"github.com/bytom/protocol/bc"
)

func calcSeed(blockHashs []*bc.Hash) []byte {
	date := []byte{}
	for _, blockHash := range blockHashs {
		date = append(date, blockHash.Bytes()...)
	}
	return crypto.Sha256(date)
}

// TODO: clean the code, now it's hard to read
func extendBytes(seed []byte, round int) []byte {
	extSeed := make([]byte, len(seed)*(round+1))
	copy(extSeed, seed)

	for i := 0; i < round; i++ {
		h := crypto.Sha256(extSeed[i*32 : (i+1)*32])
		copy(extSeed[(i+1)*32:(i+2)*32], h[:])
	}

	return extSeed
}

func calcSeedCache(seed []byte) (cache []uint32) {
	extSeed := extendBytes(seed, 3)
	xy := make([]uint32, 64)
	v := make([]uint32, 32*1024)
	for i := 0; i < 128; i++ {
		scrypt.Smix(extSeed, 1, 1024, v, xy)
		cache = append(cache, v...)
	}
	return cache
}
