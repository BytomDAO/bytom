package p2p

import (
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/go-crypto"
	cmn "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"

	cfg "github.com/bytom/config"
	"github.com/bytom/errors"
	"github.com/bytom/p2p/connection"
	"github.com/bytom/p2p/discover"
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

// MakeConnectedSwitches switches connected via arbitrary net.Conn; useful for testing
// Returns n switches, connected according to the connect func.
// If connect==Connect2Switches, the switches will be fully connected.
// initSwitch defines how the ith switch should be initialized (ie. with what reactors).
// NOTE: panics if any switch fails to start.
func MakeConnectedSwitches(cfg []*cfg.Config, n int, testDB dbm.DB, initSwitch func(*Switch) *Switch, connect func([]*Switch, int, int)) []*Switch {
	if len(cfg) != n {
		panic(errors.New("cfg number error"))
	}
	switches := make([]*Switch, n)
	for i := 0; i < n; i++ {
		switches[i] = MakeSwitch(cfg[i], testDB, initSwitch)
	}

	if err := startSwitches(switches); err != nil {
		panic(err)
	}

	for i := 0; i < n; i++ {
		for j := i; j < n; j++ {
			connect(switches, i, j)
		}
	}

	return switches
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

func (m *mockDiscv) ReadRandomNodes(buf []*discover.Node) (n int) {
	return 0
}

func MakeSwitch(cfg *cfg.Config, testdb dbm.DB, initSwitch func(*Switch) *Switch) *Switch {
	// new switch, add reactors
	// TODO: let the config be passed in?
	privKey := crypto.GenPrivKeyEd25519()
	l, listenAddr := GetListener(cfg.P2P)
	sw, err := newSwitch(cfg, new(mockDiscv), testdb, l, privKey, listenAddr)
	if err != nil {
		log.Errorf("create switch error: %s", err)
		return nil
	}
	s := initSwitch(sw)
	return s
}
