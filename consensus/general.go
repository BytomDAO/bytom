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
	initialBlockSubsidy      = uint64(1470000000000000000)
)

// define the BTM asset id, the soul asset of Bytom
var BTMAssetID = &bc.AssetID{
	V0: uint64(18446744073709551615),
	V1: uint64(18446744073709551615),
	V2: uint64(18446744073709551615),
	V3: uint64(18446744073709551615),
}

func BlockSubsidy(height uint64) uint64 {
	if height == 0 {
		return initialBlockSubsidy
	}
	return baseSubsidy >> uint(height/subsidyReductionInterval)
}

func InitBlock() []byte {
	return []byte("03010000000000000000000000000000000000000000000000000000000000000000008a9edde9ea2b40dc4e0aaee625a0b0d72c9e6fef0a70cb016ba55d86e85ebec9da87651aa15109492159980684155da19e87de0d1b37b35c1a1123770ec1dcc710aabe77607cce00fbdfb3e6cc99b32601070107008a9edde9ea2b000001012cffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8080ccdee2a69fb314010151000000")
}
