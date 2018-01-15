package consensus

import (
	"strings"

	"github.com/bytom/protocol/bc"
)

//consensus variables
const (
	// define the Max transaction size and Max block size
	MaxTxSize    = uint64(1048576)
	MaxBlockSzie = uint64(10485760)

	//config parameter for coinbase reward
	CoinbasePendingBlockNumber = uint64(6)
	subsidyReductionInterval   = uint64(560640)
	baseSubsidy                = uint64(624000000000)
	InitialBlockSubsidy        = uint64(1470000000000000000)

	// config for pow mining
	PowMinBits            = uint64(2161727821138738707)
	BlocksPerRetarget     = uint64(1024)
	TargetSecondsPerBlock = uint64(60)

	PayToWitnessPubKeyHashDataSize = 20
	PayToWitnessScriptHashDataSize = 32
)

// BTMAssetID is BTM's asset id, the soul asset of Bytom
var BTMAssetID = &bc.AssetID{
	V0: uint64(18446744073709551615),
	V1: uint64(18446744073709551615),
	V2: uint64(18446744073709551615),
	V3: uint64(18446744073709551615),
}

// BlockSubsidy calculate the coinbase rewards on given block height
func BlockSubsidy(height uint64) uint64 {
	if height == 0 {
		return InitialBlockSubsidy
	}
	return baseSubsidy >> uint(height/subsidyReductionInterval)
}

// IsBech32SegwitPrefix returns whether the prefix is a known prefix for segwit
// addresses on any default or registered network.  This is used when decoding
// an address string into a specific address type.
func IsBech32SegwitPrefix(prefix string, params *Params) bool {
	prefix = strings.ToLower(prefix)
	return prefix == params.Bech32HRPSegwit+"1"
}

// Params store the config for different network
type Params struct {
	// Name defines a human-readable identifier for the network.
	Name            string
	Bech32HRPSegwit string
}

// MainNetParams is the config for production
var MainNetParams = Params{
	Name:            "main",
	Bech32HRPSegwit: "bm",
}

// TestNetParams is the config for test-net
var TestNetParams = Params{
	Name:            "test",
	Bech32HRPSegwit: "tm",
}
