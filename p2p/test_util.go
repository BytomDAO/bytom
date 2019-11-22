package p2p

import (
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/go-crypto"
	cmn "github.com/tendermint/tmlibs/common"

	cfg "github.com/bytom/bytom/config"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/p2p/connection"
	"github.com/bytom/bytom/p2p/discover/dht"
)

//PanicOnAddPeerErr add peer error
var PanicOnAddPeerErr = false

func CreateRandomPeer(outbound bool) *Peer {
	_, netAddr := CreateRoutableAddr()
	p := &Peer{
		peerConn: &peerConn{
			outbound: outbound,
		},
		NodeInfo: &NodeInfo{
			ListenAddr: netAddr.DialString(),
		},
		mconn: &connection.MConnection{},
	}
	return p
}

func CreateRoutableAddr() (addr string, netAddr *NetAddress) {
	for {
		var err error
		addr = cmn.Fmt("%X@%v.%v.%v.%v:46656", cmn.RandBytes(20), cmn.RandInt()%256, cmn.RandInt()%256, cmn.RandInt()%256, cmn.RandInt()%256)
		netAddr, err = NewNetAddressString(addr)
		if err != nil {
			panic(err)
		}
		if netAddr.Routable() {
			break
		}
	}
	return
}

// Connect2Switches will connect switches i and j via net.Pipe()
// Blocks until a conection is established.
// NOTE: caller ensures i and j are within bounds
func Connect2Switches(switches []*Switch, i, j int) {
	switchI := switches[i]
	switchJ := switches[j]
	c1, c2 := net.Pipe()
	doneCh := make(chan struct{})
	go func() {
		err := switchI.addPeerWithConnection(c1)
		if PanicOnAddPeerErr && err != nil {
			panic(err)
		}
		doneCh <- struct{}{}
	}()
	go func() {
		err := switchJ.addPeerWithConnection(c2)
		if PanicOnAddPeerErr && err != nil {
			panic(err)
		}
		doneCh <- struct{}{}
	}()
	<-doneCh
	<-doneCh
}

func startSwitches(switches []*Switch) error {
	for _, s := range switches {
		_, err := s.Start() // start switch and reactors
		if err != nil {
			return err
		}
	}
	return nil
}

type mockDiscv struct {
}

func (m *mockDiscv) ReadRandomNodes(buf []*dht.Node) (n int) {
	return 0
}

func MakeSwitch(cfg *cfg.Config, testdb dbm.DB, privKey crypto.PrivKeyEd25519, initSwitch func(*Switch) *Switch) *Switch {
	// new switch, add reactors
	l, listenAddr := GetListener(cfg.P2P)
	cfg.P2P.LANDiscover = false
	sw, err := newSwitch(cfg, new(mockDiscv), nil, l, privKey, listenAddr)
	if err != nil {
		log.Errorf("create switch error: %s", err)
		return nil
	}
	s := initSwitch(sw)
	return s
}
