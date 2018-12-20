package p2p

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"

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

func TestNodeInfoWriteRead(t *testing.T) {
	nodeInfo := &NodeInfo{PubKey: crypto.GenPrivKeyEd25519().PubKey().Unwrap().(crypto.PubKeyEd25519), Moniker: "bytomd", Network: "mainnet", ListenAddr: "127.0.0.1:0", GenesisHash: bc.Hash{V0: 2}, BestHeight: 1024, BestHash: bc.Hash{V0: 1}, Version: "1.1.0-test", ServiceFlag: 10, Other: []string{"abc", "bcd"}}
	n, err, err1 := new(int), new(error), new(error)
	buf := new(bytes.Buffer)

	wire.WriteBinary(nodeInfo, buf, n, err)
	if *err != nil {
		t.Fatal(*err)
	}

	peerNodeInfo := new(NodeInfo)
	wire.ReadBinary(peerNodeInfo, buf, maxNodeInfoSize, new(int), err1)
	if *err1 != nil {
		t.Fatal(*err1)
	}

	if !reflect.DeepEqual(*nodeInfo, *peerNodeInfo) {
		t.Fatal("TestNodeInfoWriteRead err")
	}
}
