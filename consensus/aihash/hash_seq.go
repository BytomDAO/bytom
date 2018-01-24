package aihash

import (
	"github.com/bytom/protocol/bc"
)

var (
	//Hash128 = make(chan [128]*bc.Hash)
	Md *MiningData
)

func Observer() {
	//hash128 := <-Hash128

	//	Md = InitMiningData(hash128)
}

func Notify(hash128 [128]*bc.Hash) {
	//Hash128 <- hash
	Md = InitMiningData(hash128)
}
