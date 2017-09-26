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

	// config for pow mining
	powMinBits            = uint64(2161727821138738707)
	blocksPerRetarget     = uint64(1024)
	targetSecondsPerBlock = uint64(60)
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
	return []byte("0301000000000000000000000000000000000000000000000000000000000000000000ece090e7eb2b4078a79ed5c640a026361c4af77a37342e503cc68493229996e11dd9be38b18f5b492159980684155da19e87de0d1b37b35c1a1123770ec1dcc710aabe77607cce00b1c5a181808080802e0107010700ece090e7eb2b000001012cffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8080ccdee2a69fb314010151000000")
}
