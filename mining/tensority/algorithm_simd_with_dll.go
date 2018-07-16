// +build windows

package tensority

import "C"

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/bytom/protocol/bc"
	log "github.com/sirupsen/logrus"
)

var dllPath = fmt.Sprintf("simd_%v_%v.dll", runtime.GOOS, runtime.GOARCH)

func simdAlgorithm(bh, seed *bc.Hash) *bc.Hash {
	bhBytes := bh.Bytes()
	sdBytes := seed.Bytes()
	bhPtr := (*C.uint8_t)(unsafe.Pointer(&bhBytes[0]))
	seedPtr := (*C.uint8_t)(unsafe.Pointer(&sdBytes[0]))

	var mod = syscall.NewLazyDLL(dllPath)
	var proc = mod.NewProc("SimdTs")
	resPtr, _, _ := proc.Call(bhPtr, seedPtr)

	return bc.NewHash(*(*[32]byte)(unsafe.Pointer(resPtr)))
}

func hasSimdLib() bool {
	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		log.Warnf("SIMD lib (%v) doesn't exist, disable SIMD by default.", dllPath)
		return false
	} else {
		return true
	}
}
