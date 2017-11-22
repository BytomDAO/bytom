package consensus

import (
	"strings"

	"github.com/bytom/protocol/bc"
)

//consensus variables
const (
	// define the Max transaction size and Max block size
	MaxTxSize    = uint64(1024)
	MaxBlockSzie = uint64(16384)

	//config parameter for coinbase reward
	CoinbasePendingBlockNumber = uint64(6)
	subsidyReductionInterval   = uint64(560640)
	baseSubsidy                = uint64(624000000000)
	initialBlockSubsidy        = uint64(1470000000000000000)

	// config for pow mining
	powMinBits            = uint64(2161727821138738707)
	BlocksPerRetarget     = uint64(1024)
	targetSecondsPerBlock = uint64(60)
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
	if height == 1 {
		return initialBlockSubsidy
	}
	return baseSubsidy >> uint(height/subsidyReductionInterval)
}

// InitBlock record the byte init block
func InitBlock() []byte {
	return []byte("0301010000000000000000000000000000000000000000000000000000000000000000f9e2eaaa42d9fe9e558a9b8ef1bf366f190aacaa83bad2641ee106e9041096e4bcbcfe8bfe2b406e8760cf36f27935828aa56da40d5a8123cf186248f8fbaaae86d7a51cc8737c12553440d84371701d3d4348099f8abd59a7e7d819befa57b1de50212e5d20e3b1dc1593fcb680808080801e0107010700bcbcfe8bfe2b000001012fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8080ccdee2a69fb3140104cd57a069000000")
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
