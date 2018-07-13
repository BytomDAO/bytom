// +build windows

package tensority

import (
	"fmt"
	"os"
	"runtime"

	"github.com/bytom/protocol/bc"
	log "github.com/sirupsen/logrus"
)

var dllPath = fmt.Sprintf("simd_%v_%v.dll", runtime.GOOS, runtime.GOARCH)

func simdAlgorithm(bh, seed *bc.Hash) *bc.Hash {
	log.Warn("SIMD on windows hasn't been implemented yet, disable SIMD by default.")
	return legacyAlgorithm(bh, seed)
}

func hasSimdLib() bool {
	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		log.Warnf("SIMD lib (%v) doesn't exist, disable SIMD by default.", dllPath)
		return false
	} else {
		return true
	}
}
