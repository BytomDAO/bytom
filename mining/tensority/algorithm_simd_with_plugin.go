// +build !windows

package tensority

import (
    "fmt"
    "os"
    "runtime"

    "github.com/bytom/protocol/bc"
    log "github.com/sirupsen/logrus"
)

var pluginPath = fmt.Sprintf("simd_plugin_%v_%v.so", runtime.GOOS, runtime.GOARCH)

func simdAlgorithm(bh, seed *bc.Hash) *bc.Hash {
    if (runtime.GOOS == "linux" || runtime.GOOS == "darwin") && hasSimdLib() {
        log.Warn("SIMD hasn't been implemented yet, disable SIMD by default.")
        return legacyAlgorithm(bh, seed)
    } else {
        return legacyAlgorithm(bh, seed)
    }
}

func hasSimdLib() bool {
    if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
        log.Warnf("SIMD plugin (%v) doesn't exist, disable SIMD by default.", pluginPath)
        return false
    } else {
        return true
    }
}