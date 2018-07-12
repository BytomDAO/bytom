// +build !windows

package tensority

import (
    "fmt"
    "os"
    "runtime"

    "github.com/bytom/protocol/bc"
    log "github.com/sirupsen/logrus"
)

var libPath = fmt.Sprintf("simd_plugin_%v_%v.so", runtime.GOOS, runtime.GOARCH)

func simdAlgorithm(bh, seed *bc.Hash) *bc.Hash {
    if (runtime.GOOS == "linux" || runtime.GOOS == "darwin") && hasSimdLib() {
        log.Warn("SIMD hasn't been implemented yet, disable SIMD by default.")
        return legacyAlgorithm(bh, seed)
    } else {
        return legacyAlgorithm(bh, seed)
    }
}

func hasSimdLib() bool {
    if _, err := os.Stat(libPath); os.IsNotExist(err) {
        log.Warn("SIMD plugin doesn't exist, disable SIMD by default.")
        return false
    } else {
        return true
    }
}
