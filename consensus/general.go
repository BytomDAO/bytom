package consensus

import (
	"encoding/binary"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/crypto/ed25519/chainkd"
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
	InitBTMSupply      = 169073499178579697 + 50000000000
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

	BTMAlias = "BTM"
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
	VotePendingBlockNumber uint64

	FederationXpubs []chainkd.XPub
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
		BlockTimeInterval:      6000,
		MaxTimeOffsetMs:        2400,
		BlocksOfEpoch:          100,
		MinValidatorVoteNum:    1e14,
		VotePendingBlockNumber: 14400,
		FederationXpubs: []chainkd.XPub{
			xpub("c6dd2a72e8e598bbdb0f8314eaf1eb662341976ad26a834f1d346058adb5228e28287ee830224a7a3ff6864a7d6c6d359cf63d6a64c381dfc437a99f35098d8d"),
			xpub("920a17359fd2c4442af90d110aaac3f260f4e8868d8490508e0f2fec94dad3abd3f1b08358a61db8f545208f0e74b82d537f22181289e09e949114cd8755583d"),
			xpub("9a065ae1f84d8062cc91a7e9a710d332a618fd974fdf56dd4b543c8216fba9fb050e23abef910b98f084e83128c35a260d673258ab068423345b88e995a66dd5"),
			xpub("bf7906d2981dc8d80081d39e569ae7c2728664f5a1e718b7683b8cbf21ace99349fbc17ec3478c36aba52c6faefbd6e8975d7d2c048af2374f1587c8a9a3a3d3"),
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
		BlockTimeInterval:      6000,
		MaxTimeOffsetMs:        24000,
		BlocksOfEpoch:          100,
		MinValidatorVoteNum:    1e8,
		VotePendingBlockNumber: 10,
		FederationXpubs: []chainkd.XPub{
			xpub("7732fac62320799ff5e4eec1dc4ba7b07dc0e5a647850bf0bc34cb9aca195a05a1118b57d377947d7936156c831c87b700ed945a82cae63aff14905beb39d001"),
			xpub("08543fef8c3ca27483954f80eee6d461c307b6aa564aafaf235a4bd2740debbc71b14af78715c94cbc1d16fa84da97a3eabc5b21f003ab49882e4af7f9f00bbd"),
			xpub("0dd00fe3880c1cb5d5b0b5d03993c004e7fbe3697a47ff60c3bc12950bead964843dfe45b2bab5d01ae32fb23a4b0460049e822d7787a9a15b76d8bb9dfcec74"),
			xpub("b0584ecaefc02d3c367f280e128ec310c9f9198d44cd76b6726cd6c06c002770a1a7dc069ddd06f7a821a176931573d40e63b015ce88b6de01a61205d719567f"),
		},
	},
}

// SoloNetParams is the config for test-net
var SoloNetParams = Params{
	Name:            "solo",
	Bech32HRPSegwit: "sn",
	CasperConfig: CasperConfig{
		BlockTimeInterval:      6000,
		MaxTimeOffsetMs:        24000,
		BlocksOfEpoch:          100,
		MinValidatorVoteNum:    1e8,
		VotePendingBlockNumber: 10,
		FederationXpubs:        []chainkd.XPub{},
	},
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
