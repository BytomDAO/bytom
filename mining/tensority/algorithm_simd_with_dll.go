// +build windows

package tensority

import (
    "fmt"
    "runtime"

    "github.com/bytom/protocol/bc"
    log "github.com/sirupsen/logrus"
)

var pluginPath = fmt.Sprintf("simd_%v_%v.dll", runtime.GOOS, runtime.GOARCH)

func simdAlgorithm(bh, seed *bc.Hash) *bc.Hash {
    log.Warn("SIMD on windows hasn't been implemented yet, disable SIMD by default.")
    return legacyAlgorithm(bh, seed)
}

func hasSimdLib() bool {
    return false
}
