package consensus

import (
	"strings"

	"github.com/bytom/protocol/bc"
)

//consensus variables
const (
	// Max gas that one block contains
	MaxBlockGas = uint64(100000000)

	//config parameter for coinbase reward
	CoinbasePendingBlockNumber = uint64(6)
	subsidyReductionInterval   = uint64(560640)
	baseSubsidy                = uint64(41250000000)
	InitialBlockSubsidy        = uint64(1470000000000000000)

	// config for pow mining
	PowMinBits            = uint64(2305843009213861724)
	BlocksPerRetarget     = uint64(128)
	TargetSecondsPerBlock = uint64(60)
	SeedPerRetarget       = uint64(128)

	// MaxTimeOffsetSeconds is the maximum number of seconds a block time is allowed to be ahead of the current time
	MaxTimeOffsetSeconds = uint64(60 * 60)
	MedianTimeBlocks     = 11

	PayToWitnessPubKeyHashDataSize = 20
	PayToWitnessScriptHashDataSize = 32

	CoinbaseArbitrarySizeLimit = 128

	VMGasRate        = int64(1000)
	StorageGasRate   = int64(5)
	MaxGasAmount     = int64(100000)
	DefaultGasCredit = int64(80000)

	BTMAlias       = "BTM"
	BTMSymbol      = "BTM"
	BTMDecimals    = 8
	BTMDescription = `Bytom Official Issue`
)

// BTMAssetID is BTM's asset id, the soul asset of Bytom
var BTMAssetID = &bc.AssetID{
	V0: uint64(18446744073709551615),
	V1: uint64(18446744073709551615),
	V2: uint64(18446744073709551615),
	V3: uint64(18446744073709551615),
}

// InitialSeed is SHA3-256 of Byte[0^32]
var InitialSeed = &bc.Hash{
	V0: uint64(11412844483649490393),
	V1: uint64(4614157290180302959),
	V2: uint64(1780246333311066183),
	V3: uint64(9357197556716379726),
}

// BTMDefinitionMap is the ....
var BTMDefinitionMap = map[string]interface{}{
	"name":        BTMAlias,
	"symbol":      BTMSymbol,
	"decimals":    BTMDecimals,
	"description": BTMDescription,
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
