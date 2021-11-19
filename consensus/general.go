package consensus

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/crypto/sm2/chainkd"
	"github.com/bytom/bytom/protocol/bc"
)

// consensus variables
const (
	// Max gas that one block contains
	MaxBlockGas    = uint64(10000000)
	VMGasRate      = int64(200)
	StorageGasRate = int64(1)
	MaxGasAmount   = int64(300000)

	// These configs need add to casper config in elegant way
	MaxNumOfValidators = int(10)
	InitBTMSupply      = 169290721678579170 + 50000000000
	RewardThreshold    = 0.5
	BlockReward        = uint64(570776255)

	// config parameter for coinbase reward
	CoinbasePendingBlockNumber = uint64(10)
	MinVoteOutputAmount        = uint64(100000000)

	PayToWitnessPubKeyHashDataSize = 20
	PayToWitnessScriptHashDataSize = 32
	BCRPContractHashDataSize       = 32
	CoinbaseArbitrarySizeLimit     = 128

	BCRPRequiredBTMAmount = uint64(100000000)

	BTMAlias              = "BTM"
	defaultVotePendingNum = 302400
)

type CasperConfig struct {
	// BlockTimeInterval, milliseconds, the block time interval for producing a block
	BlockTimeInterval uint64

	// MaxTimeOffsetMs represent the max number of seconds a block time is allowed to be ahead of the current time
	MaxTimeOffsetMs uint64

	// BlocksOfEpoch represent the block num in one epoch
	BlocksOfEpoch uint64

	// MinValidatorVoteNum is the minimum vote number of become validator
	MinValidatorVoteNum uint64

	// VotePendingBlockNumber is the locked block number of vote utxo
	VotePendingBlockNums []VotePendingBlockNum

	FederationXpubs []chainkd.XPub
}

type VotePendingBlockNum struct {
	BeginBlock uint64
	EndBlock   uint64
	Num        uint64
}

// BTMAssetID is BTM's asset id, the soul asset of Bytom
var BTMAssetID = &bc.AssetID{
	V0: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
	V1: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
	V2: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
	V3: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
}

// BTMDefinitionMap is the ....
var BTMDefinitionMap = map[string]interface{}{
	"name":        BTMAlias,
	"symbol":      BTMAlias,
	"decimals":    8,
	"description": `Bytom Official Issue`,
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
	// DefaultPort defines the default peer-to-peer port for the network.
	DefaultPort string

	// DNSSeeds defines a list of DNS seeds for the network that are used
	// as one method to discover peers.
	DNSSeeds []string

	// CasperConfig defines the casper consensus parameters
	CasperConfig
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
	Bech32HRPSegwit: "bn",
	DefaultPort:     "46657",
	DNSSeeds:        []string{},
	CasperConfig: CasperConfig{
		BlockTimeInterval:   6000,
		MaxTimeOffsetMs:     3000,
		BlocksOfEpoch:       100,
		MinValidatorVoteNum: 1e14,
		VotePendingBlockNums: []VotePendingBlockNum{
			{BeginBlock: 0, EndBlock: 432000, Num: 14400},
			{BeginBlock: 432000, EndBlock: math.MaxUint64, Num: defaultVotePendingNum},
		},
		FederationXpubs: []chainkd.XPub{
			xpub("0350f12a42b1fec3945d05bcdc91cc179d3cf04fc7f401d69eff47b9a49b24ee585c6a09877b04fea18630526ead889f14e44e5896746dc3af8dc4c41fa5ad40ac"),
		},
	},
}

// TestNetParams is the config for test-net
var TestNetParams = Params{
	Name:            "test",
	Bech32HRPSegwit: "tn",
	DefaultPort:     "46656",
	DNSSeeds:        []string{},
	CasperConfig: CasperConfig{
		BlockTimeInterval:    6000,
		MaxTimeOffsetMs:      3000,
		BlocksOfEpoch:        100,
		MinValidatorVoteNum:  1e8,
		VotePendingBlockNums: []VotePendingBlockNum{{BeginBlock: 0, EndBlock: math.MaxUint64, Num: 10}},
		FederationXpubs:      []chainkd.XPub{},
	},
}

// SoloNetParams is the config for test-net
var SoloNetParams = Params{
	Name:            "solo",
	Bech32HRPSegwit: "sn",
	CasperConfig: CasperConfig{
		BlockTimeInterval:    6000,
		MaxTimeOffsetMs:      24000,
		BlocksOfEpoch:        100,
		MinValidatorVoteNum:  1e8,
		VotePendingBlockNums: []VotePendingBlockNum{{BeginBlock: 0, EndBlock: math.MaxUint64, Num: 10}},
		FederationXpubs:      []chainkd.XPub{},
	},
}

func VotePendingBlockNums(height uint64) uint64 {
	for _, pendingNum := range ActiveNetParams.VotePendingBlockNums {
		if height >= pendingNum.BeginBlock && height < pendingNum.EndBlock {
			return pendingNum.Num
		}
	}
	return defaultVotePendingNum
}

// InitActiveNetParams load the config by chain ID
func InitActiveNetParams(chainID string) error {
	var exist bool
	if ActiveNetParams, exist = NetParams[chainID]; !exist {
		return fmt.Errorf("chain_id[%v] don't exist", chainID)
	}
	return nil
}

func xpub(str string) (xpub chainkd.XPub) {
	if err := xpub.UnmarshalText([]byte(str)); err != nil {
		log.Panicf("Fail converts a string to xpub")
	}
	return xpub
}
