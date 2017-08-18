//+build pseudohsm

package main

import (
	"bytom/core"
	"bytom/core/config"
	"bytom/core/pseduokhsm"
)

func init() {
	config.BuildConfig.PseudoHSM = true
}

func enablePseudoHSM(config *config.Config) []core.RunOption {
	return []core.RunOption{core.PseudoHSM(Pseudohsm.New(config))}
}
