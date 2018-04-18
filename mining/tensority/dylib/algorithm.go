package tensority

// #cgo CFLAGS: -I.
// #cgo LDFLAGS: -ldl -L.
/*
#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <dlfcn.h>

typedef uint8_t* (*CSIMDTS_FUNC)(uint8_t*, uint8_t*, uint8_t*);
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

import(
    "unsafe"

    "github.com/bytom/protocol/bc"
)


func Hash(blockHeader, seed *bc.Hash) *bc.Hash {
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