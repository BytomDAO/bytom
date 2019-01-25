package consensus

import (
	"encoding/binary"
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
	baseSubsidy                = uint64(125000000000)
	InitialBlockSubsidy        = uint64(125000000000)

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
	V0: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
	V1: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
	V2: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
	V3: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
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
var ActiveNetParams = TestNetParams

// NetParams is the correspondence between chain_id and Params
var NetParams = map[string]Params{
	"wisdom":  TestNetParams,
	"solonet": SoloNetParams,
}

// TestNetParams is the config for test-net
var TestNetParams = Params{
	Name:            "test",
	Bech32HRPSegwit: "gm",
	Checkpoints: []Checkpoint{
		{10000, bc.NewHash([32]byte{0x2e, 0x53, 0xaf, 0x42, 0xcb, 0x7f, 0x2d, 0x1b, 0xa5, 0x10, 0xec, 0xdc, 0x1a, 0xc5, 0xa6, 0x7e, 0xd5, 0xa9, 0xb5, 0x4b, 0x39, 0x56, 0x24, 0xf9, 0x9f, 0xf4, 0xe9, 0xb0, 0xeb, 0xef, 0xdc, 0xe0})},
		{32626, bc.NewHash([32]byte{0x0c, 0xef, 0x34, 0x73, 0xc5, 0x5e, 0x0c, 0x02, 0x44, 0xbb, 0x30, 0x20, 0x5d, 0x29, 0xb6, 0x66, 0xd5, 0xa5, 0xd0, 0xc2, 0xa5, 0x48, 0x49, 0x70, 0xe9, 0x75, 0x4f, 0x38, 0xda, 0x40, 0x4e, 0x15})},
		{60104, bc.NewHash([32]byte{0xe2, 0x70, 0xea, 0x71, 0xb1, 0xee, 0x6a, 0xd9, 0x25, 0xcb, 0xcd, 0x7a, 0xef, 0x00, 0x42, 0xfe, 0xe4, 0x0d, 0x1f, 0x8f, 0x04, 0x0d, 0xce, 0x17, 0xfb, 0x19, 0x6f, 0xb2, 0x72, 0x38, 0xe0, 0xa5})},
	},
}

// SoloNetParams is the config for test-net
var SoloNetParams = Params{
	Name:            "solo",
	Bech32HRPSegwit: "sm",
	Checkpoints:     []Checkpoint{},
}
