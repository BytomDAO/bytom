// +build windows

package tensority

import (
    "github.com/bytom/protocol/bc"
    log "github.com/sirupsen/logrus"
)

var pluginPath = fmt.Sprintf("simd_plugin_%v_%v.so", runtime.GOOS, runtime.GOARCH)

func simdAlgorithm(bh, seed *bc.Hash) *bc.Hash {
    log.Warn("SIMD on windows hasn't been implemented yet, disable SIMD by default.")
    return legacyAlgorithm(bh, seed)
}

func hasSimdLib() bool {
    return false
}
