// +build simd

package cgo_algorithm

// #cgo !darwin CFLAGS: -I.
// #cgo !darwin LDFLAGS: -L. -l:./lib/cSimdTs.o -lstdc++ -lgomp -lpthread
// #cgo darwin CFLAGS: -I. -I/usr/local/opt/llvm/include
// #cgo darwin LDFLAGS: -L. -l./lib/cSimdTs.o -lstdc++ -lomp -L/usr/local/opt/llvm/lib
// #include "./lib/cSimdTs.h"
import "C"

import (
	"unsafe"

	"github.com/bytom/protocol/bc"
)

func SimdAlgorithm(bh, seed *bc.Hash) *bc.Hash {
	bhBytes := bh.Bytes()
	sdBytes := seed.Bytes()
	bhPtr := (*C.uint8_t)(unsafe.Pointer(&bhBytes[0]))
	seedPtr := (*C.uint8_t)(unsafe.Pointer(&sdBytes[0]))

	resPtr := C.SimdTs(bhPtr, seedPtr)
	resHash := bc.NewHash(*(*[32]byte)(unsafe.Pointer(resPtr)))

	return &resHash
}
