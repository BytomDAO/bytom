package p2p

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/tendermint/go-crypto"

	cfg "github.com/bytom/bytom/config"
	conn "github.com/bytom/bytom/p2p/connection"
	"github.com/bytom/bytom/version"
)

const testCh = 0x01

func TestPeerBasic(t *testing.T) {
	// simulate remote peer
	rp := &remotePeer{PrivKey: crypto.GenPrivKeyEd25519(), Config: testCfg}
	rp.Start()
	defer rp.Stop()

	p, err := createOutboundPeerAndPerformHandshake(rp.Addr(), cfg.DefaultP2PConfig())
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()
}

func TestPeerSend(t *testing.T) {
	config := testCfg

	// simulate remote peer
	rp := &remotePeer{PrivKey: crypto.GenPrivKeyEd25519(), Config: config}
	rp.Start()
	defer rp.Stop()

	p, err := createOutboundPeerAndPerformHandshake(rp.Addr(), config.P2P)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Start()
	if err != nil {
		t.Fatal(err)
	}

	defer p.Stop()
	if ok := p.CanSend(testCh); !ok {
		t.Fatal("TestPeerSend send err")
	}

	if ok := p.TrySend(testCh, []byte("test date")); !ok {
		t.Fatal("TestPeerSend try send err")
	}
}

func createOutboundPeerAndPerformHandshake(
	addr *NetAddress,
	config *cfg.P2PConfig,
) (*Peer, error) {
	chDescs := []*conn.ChannelDescriptor{
		{ID: testCh, Priority: 1},
	}
	reactorsByCh := map[byte]Reactor{testCh: NewTestReactor(chDescs, true)}
	privkey := crypto.GenPrivKeyEd25519()
	peerConfig := DefaultPeerConfig(config)
	pc, err := newOutboundPeerConn(addr, privkey, peerConfig)
	if err != nil {
		return nil, err
	}
	nodeInfo, err := pc.HandshakeTimeout(&NodeInfo{
		Moniker: "host_peer",
		Network: "testing",
		Version: "123.123.123",
	}, 5*time.Second)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	p := newPeer(pc, nodeInfo, reactorsByCh, chDescs, nil, false)
	return p, nil
}

type remotePeer struct {
	PrivKey    crypto.PrivKeyEd25519
	Config     *cfg.Config
	addr       *NetAddress
	quit       chan struct{}
	listenAddr string
}

func (rp *remotePeer) Addr() *NetAddress {
	return rp.addr
}

func (rp *remotePeer) Start() {
	if rp.listenAddr == "" {
		rp.listenAddr = "127.0.0.1:0"
	}

	l, e := net.Listen("tcp", rp.listenAddr) // any available address
	if e != nil {
		fmt.Println("net.Listen tcp :0:", e)
	}
	rp.addr = NewNetAddress(l.Addr())
	rp.quit = make(chan struct{})
	go rp.accept(l)
}

func (rp *remotePeer) Stop() {
	close(rp.quit)
}

func (rp *remotePeer) accept(l net.Listener) {
	conns := []net.Conn{}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Failed to accept conn:", err)
		}

		pc, err := newInboundPeerConn(conn, rp.PrivKey, rp.Config.P2P)
		if err != nil {
			fmt.Println("Failed to create a peer:", err)
		}

		_, err = pc.HandshakeTimeout(&NodeInfo{
			PubKey:     rp.PrivKey.PubKey().Unwrap().(crypto.PubKeyEd25519),
			Moniker:    "remote_peer",
			Network:    rp.Config.ChainID,
			Version:    version.Version,
			ListenAddr: l.Addr().String(),
		}, 5*time.Second)
		if err != nil {
			fmt.Println("Failed to perform handshake:", err)
		}
		conns = append(conns, conn)
		select {
		case <-rp.quit:
			for _, conn := range conns {
				if err := conn.Close(); err != nil {
					fmt.Println(err)
				}
			}
			return
		default:
		}
	}
}

type inboundPeer struct {
	PrivKey crypto.PrivKeyEd25519
	config  *cfg.Config
}

func (ip *inboundPeer) dial(addr *NetAddress) {
	pc, err := newOutboundPeerConn(addr, ip.PrivKey, DefaultPeerConfig(ip.config.P2P))
	if err != nil {
		fmt.Println("newOutboundPeerConn:", err)
		return
	}

	_, err = pc.HandshakeTimeout(&NodeInfo{
		PubKey:     ip.PrivKey.PubKey().Unwrap().(crypto.PubKeyEd25519),
		Moniker:    "remote_peer",
		Network:    ip.config.ChainID,
		Version:    version.Version,
		ListenAddr: addr.String(),
	}, 5*time.Second)
	if err != nil {
		fmt.Println("Failed to perform handshake:", err)
		return
	}
	time.AfterFunc(10*time.Second, pc.CloseConn)
}
