package consensus

import (
	"strings"

	"github.com/bytom/protocol/bc"
)

//consensus variables
const (
	// Max gas that one block contains
	MaxBlockGas      = uint64(10000000)
	VMGasRate        = int64(200)
	StorageGasRate   = int64(1)
	MaxGasAmount     = int64(200000)
	DefaultGasCredit = int64(30000)

	//config parameter for coinbase reward
	CoinbasePendingBlockNumber = uint64(100)
	subsidyReductionInterval   = uint64(840000)
	baseSubsidy                = uint64(41250000000)
	InitialBlockSubsidy        = uint64(140700041250000000)

	// config for pow mining
	BlocksPerRetarget     = uint64(2016)
	TargetSecondsPerBlock = uint64(150)
	SeedPerRetarget       = uint64(256)

	// MaxTimeOffsetSeconds is the maximum number of seconds a block time is allowed to be ahead of the current time
	MaxTimeOffsetSeconds = uint64(60 * 60)
	MedianTimeBlocks     = 11

	PayToWitnessPubKeyHashDataSize = 20
	PayToWitnessScriptHashDataSize = 32
	CoinbaseArbitrarySizeLimit     = 128

	BTMAlias = "BTM"
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
	"symbol":      BTMAlias,
	"decimals":    8,
	"description": `Bytom Official Issue`,
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

// Checkpoint identifies a known good point in the block chain.  Using
// checkpoints allows a few optimizations for old blocks during initial download
// and also prevents forks from old blocks.
type Checkpoint struct {
	Height uint64
	Hash   bc.Hash
}

// Params store the config for different network
type Params struct {
	// Name defines a human-readable identifier for the network.
	Name            string
	Bech32HRPSegwit string
	Checkpoints     []Checkpoint
}

// ActiveNetParams is ...
var ActiveNetParams = MainNetParams

// NetParams is the correspondence between chain_id and Params
var NetParams = map[string]Params{
	"mainnet": MainNetParams,
	"wisdom":  TestNetParams,
	"solonet": SoloNetParams,
}

// MainNetParams is the config for production
var MainNetParams = Params{
	Name:            "main",
	Bech32HRPSegwit: "bm",
	Checkpoints: []Checkpoint{
		{10000, bc.NewHash([32]byte{0x93, 0xe1, 0xeb, 0x78, 0x21, 0xd2, 0xb4, 0xad, 0x0f, 0x5b, 0x1c, 0xea, 0x82, 0xe8, 0x43, 0xad, 0x8c, 0x09, 0x9a, 0xb6, 0x5d, 0x8f, 0x70, 0xc5, 0x84, 0xca, 0xa2, 0xdd, 0xf1, 0x74, 0x65, 0x2c})},
		{20000, bc.NewHash([32]byte{0x7d, 0x38, 0x61, 0xf3, 0x2c, 0xc0, 0x03, 0x81, 0xbb, 0xcd, 0x9a, 0x37, 0x6f, 0x10, 0x5d, 0xfe, 0x6f, 0xfe, 0x2d, 0xa5, 0xea, 0x88, 0xa5, 0xe3, 0x42, 0xed, 0xa1, 0x17, 0x9b, 0xa8, 0x0b, 0x7c})},
		{30000, bc.NewHash([32]byte{0x32, 0x36, 0x06, 0xd4, 0x27, 0x2e, 0x35, 0x24, 0x46, 0x26, 0x7b, 0xe0, 0xfa, 0x48, 0x10, 0xa4, 0x3b, 0xb2, 0x40, 0xf1, 0x09, 0x51, 0x5b, 0x22, 0x9f, 0xf3, 0xc3, 0x83, 0x28, 0xaa, 0x4a, 0x00})},
		{40000, bc.NewHash([32]byte{0x7f, 0xe2, 0xde, 0x11, 0x21, 0xf3, 0xa9, 0xa0, 0xee, 0x60, 0x8d, 0x7d, 0x4b, 0xea, 0xcc, 0x33, 0xfe, 0x41, 0x25, 0xdc, 0x2f, 0x26, 0xc2, 0xf2, 0x9c, 0x07, 0x17, 0xf9, 0xe4, 0x4f, 0x9d, 0x46})},
	},
}

// TestNetParams is the config for test-net
var TestNetParams = Params{
	Name:            "test",
	Bech32HRPSegwit: "tm",
	Checkpoints: []Checkpoint{
		{1000, bc.NewHash([32]byte{0x28, 0x29, 0xc9, 0xe2, 0x19, 0x0f, 0xfe, 0xb6, 0xf2, 0x73, 0xde, 0x1a, 0xe8, 0x1f, 0xa6, 0xbc, 0x15, 0xaa, 0x08, 0xa9, 0xb8, 0x4c, 0x43, 0x25, 0x9a, 0xa0, 0x24, 0x8a, 0xd8, 0x55, 0x73, 0xca})},
		{1500, bc.NewHash([32]byte{0xc2, 0x94, 0x02, 0xd1, 0x6c, 0xa8, 0x18, 0xc4, 0x95, 0x67, 0x48, 0xb8, 0xa8, 0x42, 0xa2, 0x69, 0x8e, 0xdd, 0xb2, 0xa9, 0xb6, 0xd9, 0x30, 0xf4, 0x12, 0xdb, 0x0f, 0x1e, 0x58, 0xc9, 0x69, 0x3b})},
		{10303, bc.NewHash([32]byte{0x3e, 0x94, 0x5d, 0x35, 0x70, 0x30, 0xd4, 0x3b, 0x3d, 0xe3, 0xdd, 0x80, 0x67, 0x29, 0x9a, 0x5e, 0x09, 0xf9, 0xfb, 0x2b, 0xad, 0x5f, 0x92, 0xc8, 0x69, 0xd1, 0x42, 0x39, 0x74, 0x9a, 0xd1, 0x1c})},
	},
}

// SoloNetParams is the config for test-net
var SoloNetParams = Params{
	Name:            "solo",
	Bech32HRPSegwit: "sm",
	Checkpoints:     []Checkpoint{},
}
