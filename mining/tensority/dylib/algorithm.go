package tensority

// #cgo CFLAGS: -I.
// #cgo LDFLAGS: -ldl -L.
/*
#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <dlfcn.h>

typedef int (*CSIMDTS_FUNC)(uint8_t*, uint8_t*, uint8_t*);
#define DYLIB_CSIMDTS_PATH "./cSimdTs.dylib"

int dylib_csimdts(uint8_t blockheader[32],
                    uint8_t seed[32],
                    uint8_t res[32]){
    void *handle;
    char *error;
    CSIMDTS_FUNC csimdts_func = NULL;

    //open the dynamic lib
    handle = dlopen(DYLIB_CSIMDTS_PATH, RTLD_NOW);
    if (!handle) {
        fprintf(stderr, "%s\n", dlerror());
        return 1;
    }

    // clear previous error
    dlerror();

    // get the func
    csimdts_func = (CSIMDTS_FUNC)dlsym(handle, "SimdTs");
    if ((error = dlerror()) != NULL)  {
        fprintf(stderr, "%s\n", error);
        return 1;
    }
    csimdts_func(blockheader, seed, res);

    // close dynamic lib
    dlclose(handle);
    return 0;
}
*/
import "C"

import (
	"unsafe"

	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol/bc"
	"github.com/golang/groupcache/lru"
)

const maxAIHashCached = 64

func algorithm(blockHeader, seed *bc.Hash) *bc.Hash {
	var resAddr [32]uint8
	bhBytes := blockHeader.Bytes()
	sdBytes := seed.Bytes()

	// Get the array pointer from the corresponding slice
	bhPtr := (*C.uchar)(unsafe.Pointer(&bhBytes[0]))
	seedPtr := (*C.uchar)(unsafe.Pointer(&sdBytes[0]))
	resPtr := (*C.uchar)(unsafe.Pointer(&resAddr))

	C.dylib_csimdts(bhPtr, seedPtr, resPtr)

	resHash := bc.NewHash(*(*[32]byte)(unsafe.Pointer(resPtr)))
	return &resHash
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
