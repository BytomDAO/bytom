package p2p

import (
	"github.com/tendermint/go-crypto"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"fmt"
	cfg "github.com/bytom/config"
	dbm "github.com/bytom/database/leveldb"
	"github.com/bytom/errors"
	conn "github.com/bytom/p2p/connection"
	"github.com/davecgh/go-spew/spew"
)

var (
	testCfg *cfg.Config
)

func init() {
	testCfg = cfg.DefaultConfig()

}

/*
Each peer has one `MConnection` (multiplex connection) instance.

__multiplex__ *noun* a system or signal involving simultaneous transmission of
several messages along a single channel of communication.

Each `MConnection` handles message transmission on multiple abstract communication
`Channel`s.  Each channel has a globally unique byte id.
The byte id and the relative priorities of each `Channel` are configured upon
initialization of the connection.

There are two methods for sending messages:
	func (m MConnection) Send(chID byte, msgBytes []byte) bool {}
	func (m MConnection) TrySend(chID byte, msgBytes []byte}) bool {}

`Send(chID, msgBytes)` is a blocking call that waits until `msg` is
successfully queued for the channel with the given id byte `chID`, or until the
request times out.  The message `msg` is serialized using Go-Amino.

`TrySend(chID, msgBytes)` is a nonblocking call that returns false if the
channel's queue is full.

Inbound message bytes are handled with an onReceive callback function.
*/
type PeerMessage struct {
	PeerID  string
	Bytes   []byte
	Counter int
}

type TestReactor struct {
	BaseReactor

	mtx          sync.Mutex
	channels     []*conn.ChannelDescriptor
	logMessages  bool
	msgsCounter  int
	msgsReceived map[byte][]PeerMessage
}

func NewTestReactor(channels []*conn.ChannelDescriptor, logMessages bool) *TestReactor {
	tr := &TestReactor{
		channels:     channels,
		logMessages:  logMessages,
		msgsReceived: make(map[byte][]PeerMessage),
	}
	tr.BaseReactor = *NewBaseReactor("TestReactor", tr)

	return tr
}

// GetChannels implements Reactor
func (tr *TestReactor) GetChannels() []*conn.ChannelDescriptor {
	return tr.channels
}

// OnStart implements BaseService
func (tr *TestReactor) OnStart() error {
	tr.BaseReactor.OnStart()
	return nil
}

// OnStop implements BaseService
func (tr *TestReactor) OnStop() {
	tr.BaseReactor.OnStop()
}

// AddPeer implements Reactor by sending our state to peer.
func (tr *TestReactor) AddPeer(peer *Peer) error {
	return nil
}

// RemovePeer implements Reactor by removing peer from the pool.
func (tr *TestReactor) RemovePeer(peer *Peer, reason interface{}) {
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (tr *TestReactor) Receive(chID byte, peer *Peer, msgBytes []byte) {
	if tr.logMessages {
		tr.mtx.Lock()
		defer tr.mtx.Unlock()
		tr.msgsReceived[chID] = append(tr.msgsReceived[chID], PeerMessage{peer.ID(), msgBytes, tr.msgsCounter})
		tr.msgsCounter++
	}
}

func initSwitchFunc(sw *Switch) *Switch {
	// Make two reactors of two channels each
	sw.AddReactor("foo", NewTestReactor([]*conn.ChannelDescriptor{
		{ID: byte(0x00), Priority: 10},
		{ID: byte(0x01), Priority: 10},
	}, true))
	sw.AddReactor("bar", NewTestReactor([]*conn.ChannelDescriptor{
		{ID: byte(0x02), Priority: 10},
		{ID: byte(0x03), Priority: 10},
	}, true))

	return sw
}

//Test connect self.
func TestFiltersOutItself(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.P2P.ListenAddress = "0.0.0.0:0"

	privkeySW := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = privkeySW.String()
	fmt.Println("=== TestFiltersOutItself sw privkey:", privkeySW.String())
	s1 := MakeSwitch(&cfg, testDB, privkeySW, initSwitchFunc)

	//s1 := MakeSwitch(cfg, testDB, initSwitchFunc)
	s1.Start()
	defer s1.Stop()
	// simulate s1 having a public key and creating a remote peer with the same key
	rp := &remotePeer{PrivKey: s1.nodePrivKey, Config: testCfg}
	rp.Start()
	defer rp.Stop()
	if err = s1.DialPeerWithAddress(rp.addr); errors.Root(err) != ErrConnectSelf {
		t.Fatal(err)
	}

	//S1 dialing itself ip address
	addr := NewNetAddress(s1.listeners[0].(*DefaultListener).NetListener().Addr())

	if err := s1.DialPeerWithAddress(addr); errors.Root(err) != ErrConnectSelf {
		t.Fatal(err)
	}
}

func TestDialBannedPeer(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.P2P.ListenAddress = "0.0.0.0:0"
	privkeySW := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = privkeySW.String()
	fmt.Println("=== TestDialBannedPeer sw privkey:", privkeySW.String())
	s1 := MakeSwitch(&cfg, testDB, privkeySW, initSwitchFunc)

	//s1 := MakeSwitch(cfg, testDB, initSwitchFunc)
	s1.Start()
	defer s1.Stop()
	rp := &remotePeer{PrivKey: crypto.GenPrivKeyEd25519(), Config: testCfg}
	rp.Start()
	defer rp.Stop()
	s1.AddBannedPeer(rp.addr.IP.String())
	if err := s1.DialPeerWithAddress(rp.addr); errors.Root(err) != ErrConnectBannedPeer {
		t.Fatal(err)
	}

	s1.delBannedPeer(rp.addr.IP.String())
	if err := s1.DialPeerWithAddress(rp.addr); err != nil {
		t.Fatal(err)
	}
}

func TestDuplicateOutBoundPeer(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.P2P.ListenAddress = "0.0.0.0:0"
	privkeySW := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = privkeySW.String()
	fmt.Println("=== TestDuplicateOutBoundPeer sw privkey:", privkeySW.String())
	s1 := MakeSwitch(&cfg, testDB, privkeySW, initSwitchFunc)

	//s1 := MakeSwitch(testCfg, testDB, initSwitchFunc)
	s1.Start()
	defer s1.Stop()
	rp := &remotePeer{PrivKey: crypto.GenPrivKeyEd25519(), Config: testCfg}
	rp.Start()
	defer rp.Stop()
	if err = s1.DialPeerWithAddress(rp.addr); err != nil {
		t.Fatal(err)
	}

	if err = s1.DialPeerWithAddress(rp.addr); errors.Root(err) != ErrDuplicatePeer {
		t.Fatal(err)
	}
}

func TestDuplicateInBoundPeer(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.P2P.ListenAddress = "0.0.0.0:0"

	privkeySW := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = privkeySW.String()
	fmt.Println("=== TestDuplicateInBoundPeer sw privkey:", privkeySW.String())
	s1 := MakeSwitch(&cfg, testDB, privkeySW, initSwitchFunc)

	//s1 := MakeSwitch(cfg, testDB, initSwitchFunc)
	s1.Start()
	defer s1.Stop()

	inp := &inboundPeer{PrivKey: crypto.GenPrivKeyEd25519(), config: testCfg}
	addr := NewNetAddress(s1.listeners[0].(*DefaultListener).NetListener().Addr())
	if err != nil {
		t.Fatal(err)
	}

	if err = inp.dial(addr); err != nil {
		t.Fatal(err)
	}

	inp1 := &inboundPeer{PrivKey: inp.PrivKey, config: testCfg}

	if err = inp1.dial(addr); err != nil {
		t.Fatal(err)
	}

	if outbound, inbound, dialing := s1.NumPeers(); outbound+inbound+dialing != 1 {
		t.Fatal("TestDuplicateInBoundPeer peer size error", outbound, inbound, dialing)
	}
}

func TestAddInboundPeer(t *testing.T) {
	fmt.Println("=== TestAddInboundPeer start")
	defer fmt.Println("=== TestAddInboundPeer stop")
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.P2P.MaxNumPeers = 2
	cfg.P2P.ListenAddress = "0.0.0.0:0"
	privkeySW := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = privkeySW.String()
	fmt.Println("=== TestAddInboundPeer sw privkey:", privkeySW.String())
	s1 := MakeSwitch(&cfg, testDB, privkeySW, initSwitchFunc)
	s1.Start()
	fmt.Println("=== TestAddInboundPeer sw listen addr:", s1.nodeInfo.ListenAddr, s1.listeners[0].(*DefaultListener).NetListener().Addr())
	defer s1.Stop()

	cfginp := *testCfg
	privkey := crypto.GenPrivKeyEd25519()
	fmt.Println("=== TestAddInboundPeer inpeer privkey:", privkey.String())

	cfginp.P2P.PrivateKey = privkey.String()
	inp := &inboundPeer{PrivKey: privkey, config: &cfginp}
	addr := NewNetAddress(s1.listeners[0].(*DefaultListener).NetListener().Addr())
	if err != nil {
		t.Fatal(err)
	}

	if err := inp.dial(addr); err != nil {
		t.Fatal(err)
	}

	cfgrp := *testCfg
	privkeyrp := crypto.GenPrivKeyEd25519()
	fmt.Println("=== TestAddInboundPeer remote peer privkey:", privkeyrp.String())

	cfginp.P2P.PrivateKey = privkeyrp.String()

	rp := &remotePeer{PrivKey: privkeyrp, Config: &cfgrp}
	rp.Start()
	defer rp.Stop()
	fmt.Println("=== TestAddInboundPeer remote peer addr:", rp.addr)
	if err := s1.DialPeerWithAddress(rp.addr); err != nil {
		t.Fatal(err)
	}

	if outbound, inbound, dialing := s1.NumPeers(); outbound+inbound+dialing != 2 {
		t.Fatal("TestAddInboundPeer peer size error")
	}
	cfginp2 := *testCfg
	privkeyinp2 := crypto.GenPrivKeyEd25519()
	fmt.Println("=== TestAddInboundPeer inpeer2 privkey:", privkeyinp2.String())
	cfginp2.P2P.PrivateKey = privkeyinp2.String()
	inp2 := &inboundPeer{PrivKey: privkeyinp2, config: &cfginp2}

	if err := inp2.dial(addr); err == nil {
		t.Fatal("TestAddInboundPeer MaxNumPeers limit error")
	}
}

func TestStopPeer(t *testing.T) {
	fmt.Println("=== TestStopPeer start")
	defer fmt.Println("=== TestStopPeer stop")
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.P2P.MaxNumPeers = 2
	cfg.P2P.ListenAddress = "0.0.0.0:0"
	privkeySW := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = privkeySW.String()
	fmt.Println("=== TestStopPeer sw privkey:", privkeySW.String())
	s1 := MakeSwitch(&cfg, testDB, privkeySW, initSwitchFunc)

	//s1 := MakeSwitch(&cfg, testDB, initSwitchFunc)
	s1.Start()
	fmt.Println("=== TestStopPeer sw listen addr:", s1.nodeInfo.ListenAddr, s1.listeners[0].(*DefaultListener).NetListener().Addr())

	defer s1.Stop()

	cfginp := *testCfg
	privkeyinp := crypto.GenPrivKeyEd25519()
	fmt.Println("=== TestStopPeer inpeer privkey:", privkeyinp.String())
	cfginp.P2P.PrivateKey = privkeyinp.String()
	inp := &inboundPeer{PrivKey: privkeyinp, config: testCfg}
	addr := NewNetAddress(s1.listeners[0].(*DefaultListener).NetListener().Addr())
	if err != nil {
		t.Fatal(err)
	}

	if err := inp.dial(addr); err != nil {
		t.Fatal(err)
	}
	cfgrp := *testCfg
	privkeyrp := crypto.GenPrivKeyEd25519()
	cfginp.P2P.PrivateKey = privkeyrp.String()
	fmt.Println("=== TestStopPeer remote peer privkey:", privkeyrp.String())

	rp := &remotePeer{PrivKey: privkeyrp, Config: &cfgrp}
	rp.Start()
	defer rp.Stop()
	fmt.Println("=== TestStopPeer remote peer addr:", rp.addr)
	if err := s1.DialPeerWithAddress(rp.addr); err != nil {
		t.Fatal(err)
	}

	if outbound, inbound, dialing := s1.NumPeers(); outbound+inbound+dialing != 2 {
		t.Fatalf("want 2 got %s", spew.Sdump(s1.peers.list))
		t.Fatal("TestStopPeer peer size error")
	}

	s1.StopPeerGracefully(s1.peers.list[0].Key)
	if outbound, inbound, dialing := s1.NumPeers(); outbound+inbound+dialing != 1 {
		t.Fatalf("want 1 got %s", spew.Sdump(s1.peers.list))
		t.Fatal("TestStopPeer peer size error")
	}

	s1.StopPeerForError(s1.peers.list[0], "stop for test")
	if outbound, inbound, dialing := s1.NumPeers(); outbound+inbound+dialing != 0 {
		t.Fatalf("want 0 got %s", spew.Sdump(s1.peers.list))
		t.Fatal("TestStopPeer peer size error")
	}
}
