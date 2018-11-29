package p2p

import (
	"testing"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

func mockCompatibleWithFalse(remoteVerStr string) (bool, error) {
	return false, nil
}

func mockCompatibleWithTrue(remoteVerStr string) (bool, error) {
	return true, nil
}

func TestCompatibleWith(t *testing.T) {
	nodeInfo := &NodeInfo{Network: "testnet", GenesisHash: bc.Hash{V0: 1}}

	cases := []struct {
		other                 *NodeInfo
		versionCompatibleWith VersionCompatibleWith
		err                   error
	}{
		{other: &NodeInfo{Network: "mainnet", GenesisHash: bc.Hash{V0: 0}}, versionCompatibleWith: mockCompatibleWithTrue, err: errDiffNetwork},
		{other: &NodeInfo{Network: "testnet", GenesisHash: bc.Hash{V0: 1}}, versionCompatibleWith: mockCompatibleWithTrue, err: nil},
		{other: &NodeInfo{Network: "testnet", GenesisHash: bc.Hash{V0: 2}}, versionCompatibleWith: mockCompatibleWithTrue, err: errDiffGenesis},
		{other: &NodeInfo{Network: "testnet", GenesisHash: bc.Hash{V0: 1}}, versionCompatibleWith: mockCompatibleWithFalse, err: errDiffMajorVersion},
	}

	for _, c := range cases {
		if err := nodeInfo.compatibleWith(c.other, c.versionCompatibleWith); errors.Root(err) != c.err {
			t.Fatalf("node info compatible test err want:%s result:%s", c.err, err)
		}
	}
}
