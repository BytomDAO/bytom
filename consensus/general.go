package consensus

import (
	"github.com/bytom/protocol/bc"
)

const (
	// define the Max transaction size and Max block size
	MaxTxSize    = uint64(1024)
	MaxBlockSzie = uint64(16384)

	//config parameter for coinbase reward
	subsidyReductionInterval = uint64(560640)
	baseSubsidy              = uint64(624000000000)
)

// define the BTM asset id, the soul asset of Bytom
var BTMAssetID = &bc.AssetID{
	V0: uint64(18446744073709551615),
	V1: uint64(18446744073709551615),
	V2: uint64(18446744073709551615),
	V3: uint64(18446744073709551615),
}

func BlockSubsidy(height uint64) uint64 {
	return baseSubsidy >> uint(height/subsidyReductionInterval)
}
