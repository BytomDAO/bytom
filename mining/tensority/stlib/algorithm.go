package tensority

// #cgo CFLAGS: -I.
// #cgo LDFLAGS: -L./stlib/ -l:cSimdTs.o -lstdc++ -lgomp
// #include "./stlib/cSimdTs.h"
import "C"

import(
    "unsafe"

    "github.com/bytom/protocol/bc"
)

func Hash(blockHeader, seed *bc.Hash) *bc.Hash {
    bhBytes := blockHeader.Bytes()
    sdBytes := seed.Bytes()

    // Get thearray pointer from the corresponding slice
    bhPtr := (*C.uchar)(unsafe.Pointer(&bhBytes[0]))
    seedPtr := (*C.uchar)(unsafe.Pointer(&sdBytes[0]))

    resPtr := C.SimdTs(bhPtr, seedPtr)
    
    res := bc.NewHash(*(*[32]byte)(unsafe.Pointer(resPtr)))
    return &res
}