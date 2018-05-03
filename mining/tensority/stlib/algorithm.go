package tensority

// #cgo !darwin CFLAGS: -I.
// #cgo !darwin LDFLAGS: -L./stlib/ -l:cSimdTs.o -lstdc++ -lgomp
// #cgo darwin CFLAGS: -I. -I/usr/local/opt/llvm/include
// #cgo darwin LDFLAGS: -L./stlib/ -lcSimdTs.o -lstdc++ -lomp -L/usr/local/opt/llvm/lib
// #include "./stlib/cSimdTs.h"
import "C"

import (
	"unsafe"

	"github.com/golang/groupcache/lru"

	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol/bc"
)

const maxAIHashCached = 64

func algorithm(blockHeader, seed *bc.Hash) *bc.Hash {
	bhBytes := blockHeader.Bytes()
	sdBytes := seed.Bytes()

	// Get thearray pointer from the corresponding slice
	bhPtr := (*C.uchar)(unsafe.Pointer(&bhBytes[0]))
	seedPtr := (*C.uchar)(unsafe.Pointer(&sdBytes[0]))

	resPtr := C.SimdTs(bhPtr, seedPtr)

	res := bc.NewHash(*(*[32]byte)(unsafe.Pointer(resPtr)))
	return &res
}

func calcCacheKey(hash, seed *bc.Hash) *bc.Hash {
	var b32 [32]byte
	sha3pool.Sum256(b32[:], append(hash.Bytes(), seed.Bytes()...))
	key := bc.NewHash(b32)
	return &key
}

// Cache is create for cache the tensority result
type Cache struct {
	lruCache *lru.Cache
}

// NewCache create a cache struct
func NewCache() *Cache {
	return &Cache{lruCache: lru.New(maxAIHashCached)}
}

// AddCache is used for add tensority calculate result
func (a *Cache) AddCache(hash, seed, result *bc.Hash) {
	key := calcCacheKey(hash, seed)
	a.lruCache.Add(*key, result)
}

// Hash is the real entry for call tensority algorithm
func (a *Cache) Hash(hash, seed *bc.Hash) *bc.Hash {
	key := calcCacheKey(hash, seed)
	if v, ok := a.lruCache.Get(*key); ok {
		return v.(*bc.Hash)
	}
	return algorithm(hash, seed)
}

// AIHash is created for let different package share same cache
var AIHash = NewCache()
