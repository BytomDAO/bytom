package consensus

import (
	"strings"
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
	BlocksPerRetarget     = uint64(1024)
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
	if height == 1 {
		return initialBlockSubsidy
	}
	return baseSubsidy >> uint(height/subsidyReductionInterval)
}

func InitBlock() []byte {
	return []byte("0301010000000000000000000000000000000000000000000000000000000000000000cecccaebf42b406b03545ed2b38a578e5e6b0796d4ebdd8a6dd72210873fcc026c7319de578ffc492159980684155da19e87de0d1b37b35c1a1123770ec1dcc710aabe77607cced7bb1993fcb680808080801e0107010700cecccaebf42b000001012cffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8080ccdee2a69fb314010151000000")
}


// IsBech32SegwitPrefix returns whether the prefix is a known prefix for segwit
// addresses on any default or registered network.  This is used when decoding
// an address string into a specific address type.
func IsBech32SegwitPrefix(prefix string,  params *Params) bool {
	prefix = strings.ToLower(prefix)
	if prefix == params.Bech32HRPSegwit+"1"  {
		return true
	} else {
		return false
	}
}


type Params struct {
	// Name defines a human-readable identifier for the network.
	Name string

	Bech32HRPSegwit string

}


var MainNetParams = Params {
	Name: "main",
	Bech32HRPSegwit: "bm",
}


var TestNetParams = Params {
	Name: "test",
	Bech32HRPSegwit: "tm",
}

