// +build linux darwin

package tensority

import (
	"fmt"
	"os"
	"plugin"
	"runtime"

	"github.com/bytom/protocol/bc"
	log "github.com/sirupsen/logrus"
)

var pluginPath = fmt.Sprintf("simd_plugin_%v_%v.so", runtime.GOOS, runtime.GOARCH)

func simdAlgorithm(bh, seed *bc.Hash) *bc.Hash {
	if (runtime.GOOS == "linux" || runtime.GOOS == "darwin") && hasSimdLib() {
		// init
		p, err := plugin.Open(pluginPath)
		if err != nil {
			log.Warnf("SIMD plugin (%v) open error, disable SIMD by default.", pluginPath)
			return legacyAlgorithm(bh, seed)
		}
		bh_v_sym, err := p.Lookup("BH")
		if err != nil {
			log.Warnf("BH symbol lookup error, disable SIMD by default.")
			return legacyAlgorithm(bh, seed)
		}
		seed_v_sym, err := p.Lookup("SEED")
		if err != nil {
			log.Warnf("SEED symbol lookup error, disable SIMD by default.")
			return legacyAlgorithm(bh, seed)
		}
		res_v_sym, err := p.Lookup("RES")
		if err != nil {
			log.Warnf("RES symbol lookup error, disable SIMD by default.")
			return legacyAlgorithm(bh, seed)
		}
		cgoAlgorithm_f_sym, err := p.Lookup("CgoAlgorithm")
		if err != nil {
			log.Warnf("CgoAlgorithm symbol lookup error, disable SIMD by default.")
			return legacyAlgorithm(bh, seed)
		}
		*bh_v_sym.(*bc.Hash) = *bh
		*seed_v_sym.(*bc.Hash) = *seed

		// invoke the func in the plugin
		cgoAlgorithm_f_sym.(func())()

		return res_v_sym.(*bc.Hash)
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
