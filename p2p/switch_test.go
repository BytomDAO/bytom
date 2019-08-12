package p2p

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/tendermint/go-crypto"

	cfg "github.com/bytom/config"
	dbm "github.com/bytom/database/leveldb"
	"github.com/bytom/errors"
	conn "github.com/bytom/p2p/connection"
	"github.com/bytom/p2p/security"
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
	t.Skip("due to fail on mac")
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.DBPath = dirPath
	cfg.P2P.ListenAddress = "127.0.1.1:0"
	swPrivKey := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = swPrivKey.String()
	s1 := MakeSwitch(&cfg, testDB, swPrivKey, initSwitchFunc)
	s1.Start()
	defer s1.Stop()

	rmdirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(rmdirPath)

	// simulate s1 having a public key and creating a remote peer with the same key
	rpCfg := *testCfg
	rpCfg.DBPath = rmdirPath
	rp := &remotePeer{PrivKey: s1.nodePrivKey, Config: &rpCfg}
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
	t.Skip("due to fail on mac")
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.DBPath = dirPath
	cfg.P2P.ListenAddress = "127.0.1.1:0"
	swPrivKey := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = swPrivKey.String()
	s1 := MakeSwitch(&cfg, testDB, swPrivKey, initSwitchFunc)
	s1.Start()
	defer s1.Stop()

	rmdirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(rmdirPath)

	rpCfg := *testCfg
	rpCfg.DBPath = rmdirPath
	rp := &remotePeer{PrivKey: crypto.GenPrivKeyEd25519(), Config: &rpCfg}
	rp.Start()
	defer rp.Stop()
	for {
		if ok := s1.security.IsBanned(rp.addr.IP.String(), security.LevelMsgIllegal, "test"); ok {
			break
		}
	}
	if err := s1.DialPeerWithAddress(rp.addr); errors.Root(err) != security.ErrConnectBannedPeer {
		t.Fatal(err)
	}
}

func TestDuplicateOutBoundPeer(t *testing.T) {
	t.Skip("due to fail on mac")
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.DBPath = dirPath
	cfg.P2P.ListenAddress = "127.0.1.1:0"
	swPrivKey := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = swPrivKey.String()
	s1 := MakeSwitch(&cfg, testDB, swPrivKey, initSwitchFunc)
	s1.Start()
	defer s1.Stop()

	rmdirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(rmdirPath)

	rpCfg := *testCfg
	rp := &remotePeer{PrivKey: crypto.GenPrivKeyEd25519(), Config: &rpCfg}
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
	t.Skip("due to fail on mac")
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.DBPath = dirPath
	cfg.P2P.ListenAddress = "127.0.1.1:0"
	swPrivKey := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = swPrivKey.String()
	s1 := MakeSwitch(&cfg, testDB, swPrivKey, initSwitchFunc)
	s1.Start()
	defer s1.Stop()

	inpCfg := *testCfg
	inp := &inboundPeer{PrivKey: crypto.GenPrivKeyEd25519(), config: &inpCfg}
	addr := NewNetAddress(s1.listeners[0].(*DefaultListener).NetListener().Addr())
	if err != nil {
		t.Fatal(err)
	}
	go inp.dial(addr)

	inp1Cfg := *testCfg
	inp1 := &inboundPeer{PrivKey: inp.PrivKey, config: &inp1Cfg}
	go inp1.dial(addr)

	time.Sleep(1 * time.Second)
	if _, outbound, inbound, dialing := s1.NumPeers(); outbound+inbound+dialing != 1 {
		t.Fatal("TestDuplicateInBoundPeer peer size error want 1, got:", outbound, inbound, dialing, spew.Sdump(s1.peers.lookup))
	}
}

func TestAddInboundPeer(t *testing.T) {
	t.Skip("due to fail on mac")
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.DBPath = dirPath
	cfg.P2P.MaxNumPeers = 2
	cfg.P2P.ListenAddress = "127.0.1.1:0"
	swPrivKey := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = swPrivKey.String()
	s1 := MakeSwitch(&cfg, testDB, swPrivKey, initSwitchFunc)
	s1.Start()
	defer s1.Stop()

	inpCfg := *testCfg
	inpPrivKey := crypto.GenPrivKeyEd25519()
	inpCfg.P2P.PrivateKey = inpPrivKey.String()
	inp := &inboundPeer{PrivKey: inpPrivKey, config: &inpCfg}
	addr := NewNetAddress(s1.listeners[0].(*DefaultListener).NetListener().Addr())
	if err != nil {
		t.Fatal(err)
	}
	go inp.dial(addr)

	rpCfg := *testCfg
	rpPrivKey := crypto.GenPrivKeyEd25519()
	rpCfg.P2P.PrivateKey = rpPrivKey.String()
	rp := &remotePeer{PrivKey: rpPrivKey, Config: &rpCfg}
	rp.Start()
	defer rp.Stop()

	if err := s1.DialPeerWithAddress(rp.addr); err != nil {
		t.Fatal(err)
	}

	inp2Cfg := *testCfg
	inp2PrivKey := crypto.GenPrivKeyEd25519()
	inp2Cfg.P2P.PrivateKey = inp2PrivKey.String()
	inp2 := &inboundPeer{PrivKey: inp2PrivKey, config: &inp2Cfg}

	go inp2.dial(addr)

	time.Sleep(1 * time.Second)
	if _, outbound, inbound, dialing := s1.NumPeers(); outbound+inbound+dialing != 2 {
		t.Fatal("TestAddInboundPeer peer size error want 2 got:", spew.Sdump(s1.peers.lookup))
	}
}

func TestStopPeer(t *testing.T) {
	t.Skip("due to fail on mac")
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	cfg := *testCfg
	cfg.DBPath = dirPath
	cfg.P2P.MaxNumPeers = 2
	cfg.P2P.ListenAddress = "127.0.1.1:0"
	swPrivKey := crypto.GenPrivKeyEd25519()
	cfg.P2P.PrivateKey = swPrivKey.String()
	s1 := MakeSwitch(&cfg, testDB, swPrivKey, initSwitchFunc)
	s1.Start()
	defer s1.Stop()

	inpCfg := *testCfg
	inpPrivKey := crypto.GenPrivKeyEd25519()
	inpCfg.P2P.PrivateKey = inpPrivKey.String()
	inp := &inboundPeer{PrivKey: inpPrivKey, config: &inpCfg}
	addr := NewNetAddress(s1.listeners[0].(*DefaultListener).NetListener().Addr())
	if err != nil {
		t.Fatal(err)
	}
	go inp.dial(addr)

	rpCfg := *testCfg
	rpPrivKey := crypto.GenPrivKeyEd25519()
	rpCfg.P2P.PrivateKey = rpPrivKey.String()
	rp := &remotePeer{PrivKey: rpPrivKey, Config: &rpCfg}
	rp.Start()
	defer rp.Stop()

	if err := s1.DialPeerWithAddress(rp.addr); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
	if _, outbound, inbound, dialing := s1.NumPeers(); outbound+inbound+dialing != 2 {
		t.Fatal("TestStopPeer peer size error want 2,got:", spew.Sdump(s1.peers.lookup))
	}

	s1.StopPeerGracefully(s1.peers.list[0].Key)
	if _, outbound, inbound, dialing := s1.NumPeers(); outbound+inbound+dialing != 1 {
		t.Fatal("TestStopPeer peer size error,want 1,got:", spew.Sdump(s1.peers.lookup))
	}

	s1.StopPeerForError(s1.peers.list[0], "stop for test")
	if _, outbound, inbound, dialing := s1.NumPeers(); outbound+inbound+dialing != 0 {
		t.Fatal("TestStopPeer peer size error,want 0, got:", spew.Sdump(s1.peers.lookup))
	}
}
