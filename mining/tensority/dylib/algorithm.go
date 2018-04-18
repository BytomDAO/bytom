package tensority

// #cgo CFLAGS: -I.
// #cgo LDFLAGS: -ldl -L.
/*
#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <dlfcn.h>

typedef uint8_t* (*CSIMDTS_FUNC)(uint8_t*, uint8_t*, uint8_t*);
#define LINUX_LIB_CSIMDTS_PATH "./cSimdTs.dylib"

int linux_csimdts(uint8_t blockheader[32],
                    uint8_t seed[32],
                    uint8_t res[32]){
    void *handle;
    char *error;
    CSIMDTS_FUNC csimdts_func = NULL;

    //open the dynamic lib
    handle = dlopen(LINUX_LIB_CSIMDTS_PATH, RTLD_NOW);
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

// func Hash(blockHeader [32]uint8, seed [32]uint8) [32]uint8 {
//     var res [32]uint8

//     bhPtr := (*C.uchar)(unsafe.Pointer(&blockHeader))
//     seedPtr := (*C.uchar)(unsafe.Pointer(&seed))
//     resPtr := (*C.uchar)(unsafe.Pointer(&res))

//     C.linux_csimdts(bhPtr, seedPtr, resPtr)

//     res = *(*[32]uint8)(unsafe.Pointer(resPtr))
//     return res
// }


func Hash(blockHeader, seed *bc.Hash) *bc.Hash {
    var resAddr [32]uint8
    bhBytes := blockHeader.Bytes()
    sdBytes := seed.Bytes()

    // Get thearray pointer from the corresponding slice
    bhPtr := (*C.uchar)(unsafe.Pointer(&bhBytes[0]))
    seedPtr := (*C.uchar)(unsafe.Pointer(&sdBytes[0]))
    resPtr := (*C.uchar)(unsafe.Pointer(&resAddr))

    C.linux_csimdts(bhPtr, seedPtr, resPtr)
    
    resHash := bc.NewHash(*(*[32]byte)(unsafe.Pointer(resPtr)))
    return &resHash
}