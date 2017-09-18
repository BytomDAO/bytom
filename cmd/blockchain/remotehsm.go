//+build remotehsm

package main

import (
	"bytom/core"
	"bytom/core/config"
)

func init() {
	config.BuildConfig.PseudoHSM = false
}

func enableHSM(config *config.Config) []core.RunOption {
	return []core.RunOption{core.RemoteHSM(Remotehsm.New(config))}
}
