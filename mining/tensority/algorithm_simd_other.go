// +build !windows,!linux,!darwin

package tensority

import (
    "github.com/bytom/protocol/bc"
    log "github.com/sirupsen/logrus"
)

func simdAlgorithm(bh, seed *bc.Hash) *bc.Hash {
    log.Warn("SIMD on other OS hasn't been implemented yet, disable SIMD by default.")
    return legacyAlgorithm(bh, seed)
}
