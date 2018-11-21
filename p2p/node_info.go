package p2p

import (
	"fmt"
	"net"

	"github.com/tendermint/go-crypto"

	"github.com/bytom/consensus"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
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
	GenesisHash bc.Hash               `json:"genesis_hash"`
	BestHeight  uint64                `json:"best_height"`
	BestHash    bc.Hash               `json:"best_hash"`
	Version     string                `json:"version"` // major.minor.revision
	ServiceFlag consensus.ServiceFlag `json:"service_flag"`
	Other       []string              `json:"other"` // other application specific data
}

type VersionCompatibleWith func(remoteVerStr string) (bool, error)

// CompatibleWith checks if two NodeInfo are compatible with eachother.
// CONTRACT: two nodes are compatible if the major version matches and network match
func (info *NodeInfo) CompatibleWith(other *NodeInfo, versionCompatibleWith VersionCompatibleWith) error {
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

//ListenHost peer listener ip address
func (info *NodeInfo) ListenHost() string {
	host, _, _ := net.SplitHostPort(info.ListenAddr)
	return host
}

//RemoteAddrHost peer external ip address
func (info *NodeInfo) RemoteAddrHost() string {
	host, _, _ := net.SplitHostPort(info.RemoteAddr)
	return host
}

//String representation
func (info NodeInfo) String() string {
	return fmt.Sprintf("NodeInfo{pk: %v, moniker: %v, network: %v [listen %v], version: %v service: %v bestHeight: %v, bestHash: %v}", info.PubKey, info.Moniker, info.Network, info.ListenAddr, info.Version, info.ServiceFlag, info.BestHeight, info.BestHash)
}
