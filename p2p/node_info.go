package p2p

import (
	"fmt"
	"net"

	"github.com/tendermint/go-crypto"

	cfg "github.com/bytom/config"
	"github.com/bytom/consensus"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/version"
)

const maxNodeInfoSize = 10240 // 10Kb

var (
	errDiffMajorVersion = errors.New("Peer is on a different major version.")
	errDiffNetwork      = errors.New("Peer is on a different network.")
	errDiffGenesis      = errors.New("Peer has different genesis.")
)

//NodeInfo peer node info
type NodeInfo struct {
	PubKey      crypto.PubKeyEd25519  `json:"pub_key"`
	Moniker     string                `json:"moniker"`
	Network     string                `json:"network"`
	RemoteAddr  string                `json:"remote_addr"`
	ListenAddr  string                `json:"listen_addr"`
	Version     string                `json:"version"` // major.minor.revision
	Other       []string              `json:"other"`   // other application specific data
	GenesisHash bc.Hash               `json:"genesis_hash"`
	BestHeight  uint64                `json:"best_height"`
	BestHash    bc.Hash               `json:"best_hash"`
	ServiceFlag consensus.ServiceFlag `json:"service_flag"`
}

type VersionCompatibleWith func(remoteVerStr string) (bool, error)

func newNodeInfo(config *cfg.Config, privKey crypto.PrivKeyEd25519, genesisHash bc.Hash, bestBlockHeader types.BlockHeader, listenAddr string) *NodeInfo {
	return &NodeInfo{
		PubKey:      privKey.PubKey().Unwrap().(crypto.PubKeyEd25519),
		Moniker:     config.Moniker,
		Network:     config.ChainID,
		ListenAddr:  listenAddr,
		Version:     version.Version,
		GenesisHash: genesisHash,
		BestHeight:  bestBlockHeader.Height,
		BestHash:    bestBlockHeader.Hash(),
		ServiceFlag: consensus.DefaultServices,
	}
}

// CompatibleWith checks if two NodeInfo are compatible with eachother.
// CONTRACT: two nodes are compatible if the major version matches and network match
func (info *NodeInfo) compatibleWith(other *NodeInfo, versionCompatibleWith VersionCompatibleWith) error {
	compatible, err := versionCompatibleWith(other.Version)
	if err != nil {
		return err
	}

	if !compatible {
		return errors.Wrapf(errDiffMajorVersion, "Peer version: %v, node version: %v", other.Version, info.Version)
	}

	if info.Network != other.Network {
		return errors.Wrapf(errDiffNetwork, "Peer network: %v, node network: %v", other.Network, info.Network)
	}

	if info.GenesisHash != other.GenesisHash {
		return errors.Wrapf(errDiffGenesis, "Peer genesis hash: %x, node genesis hash: %x", other.GenesisHash, info.GenesisHash)
	}

	return nil
}

func (info *NodeInfo) getPubkey() crypto.PubKeyEd25519 {
	return info.PubKey
}

//ListenHost peer listener ip address
func (info *NodeInfo) listenHost() string {
	host, _, _ := net.SplitHostPort(info.ListenAddr)
	return host
}

//RemoteAddrHost peer external ip address
func (info *NodeInfo) remoteAddrHost() string {
	host, _, _ := net.SplitHostPort(info.RemoteAddr)
	return host
}

func (info *NodeInfo) setPubkey(pubkey crypto.PubKeyEd25519) {
	info.PubKey = pubkey
}

//String representation
func (info *NodeInfo) String() string {
	return fmt.Sprintf("NodeInfo{pk: %v, moniker: %v, network: %v [listen %v], version: %v service: %v genesisHash:%v bestHeight: %v bestHash: %v}", info.PubKey, info.Moniker, info.Network, info.ListenAddr, info.Version, info.ServiceFlag, info.BestHash.String(), info.BestHeight, info.BestHash.String())
}

func (info *NodeInfo) updateBestHeight(bestHeight uint64, bestHash bc.Hash) {
	info.BestHeight = bestHeight
	info.BestHash = bestHash
}

func (info *NodeInfo) version() string {
	return info.Version
}
