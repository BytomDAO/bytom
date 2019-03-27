package p2p

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"

	cfg "github.com/bytom/config"
	"github.com/bytom/errors"
	"github.com/bytom/p2p/upnp"
)

const (
	numBufferedConnections = 10
	defaultExternalPort    = 8770
	tryListenTimes         = 5
)

//Listener subset of the methods of DefaultListener
type Listener interface {
	Connections() <-chan net.Conn
	InternalAddress() *NetAddress
	ExternalAddress() *NetAddress
	String() string
	Stop() bool
}

// Defaults to tcp
func protocolAndAddress(listenAddr string) (string, string) {
	p, address := "tcp", listenAddr
	parts := strings.SplitN(address, "://", 2)
	if len(parts) == 2 {
		p, address = parts[0], parts[1]
	}
	return p, address
}

// GetListener get listener and listen address.
func GetListener(config *cfg.P2PConfig) (Listener, string) {
	p, address := protocolAndAddress(config.ListenAddress)
	l, listenerStatus := NewDefaultListener(p, address, config.SkipUPNP)

	// We assume that the rpcListener has the same ExternalAddress.
	// This is probably true because both P2P and RPC listeners use UPnP,
	// except of course if the rpc is only bound to localhost
	if listenerStatus {
		return l, cmn.Fmt("%v:%v", l.ExternalAddress().IP.String(), l.ExternalAddress().Port)
	}

	return l, cmn.Fmt("%v:%v", l.InternalAddress().IP.String(), l.InternalAddress().Port)
}

//getUPNPExternalAddress UPNP external address discovery & port mapping
func getUPNPExternalAddress(externalPort, internalPort int) (*NetAddress, error) {
	nat, err := upnp.Discover()
	if err != nil {
		return nil, errors.Wrap(err, "could not perform UPNP discover")
	}

	ext, err := nat.GetExternalAddress()
	if err != nil {
		return nil, errors.Wrap(err, "could not perform UPNP external address")
	}

	if externalPort == 0 {
		externalPort = defaultExternalPort
	}
	externalPort, err = nat.AddPortMapping("tcp", externalPort, internalPort, "bytomd tcp", 0)
	if err != nil {
		return nil, errors.Wrap(err, "could not add tcp UPNP port mapping")
	}
	externalPort, err = nat.AddPortMapping("udp", externalPort, internalPort, "bytomd udp", 0)
	if err != nil {
		return nil, errors.Wrap(err, "could not add udp UPNP port mapping")
	}
	return NewNetAddressIPPort(ext, uint16(externalPort)), nil
}

func splitHostPort(addr string) (host string, port int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		cmn.PanicSanity(err)
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		cmn.PanicSanity(err)
	}
	return host, port
}

//DefaultListener Implements bytomd server Listener
type DefaultListener struct {
	cmn.BaseService

	listener    net.Listener
	intAddr     *NetAddress
	extAddr     *NetAddress
	connections chan net.Conn
}

//NewDefaultListener create a default listener
func NewDefaultListener(protocol string, lAddr string, skipUPNP bool) (Listener, bool) {
	// Local listen IP & port
	lAddrIP, lAddrPort := splitHostPort(lAddr)

	listener, err := net.Listen(protocol, lAddr)
	for i := 0; i < tryListenTimes && err != nil; i++ {
		time.Sleep(time.Second * 1)
		listener, err = net.Listen(protocol, lAddr)
	}
	if err != nil {
		log.Panic(err)
	}

	intAddr, err := NewNetAddressString(lAddr)
	if err != nil {
		log.Panic(err)
	}

	// Actual listener local IP & port
	listenerIP, listenerPort := splitHostPort(listener.Addr().String())
	log.Info("Local listener", " ip:", listenerIP, " port:", listenerPort)

	// Determine external address...
	var extAddr *NetAddress
	var upnpMap bool
	if !skipUPNP && (lAddrIP == "" || lAddrIP == "0.0.0.0") {
		extAddr, err = getUPNPExternalAddress(lAddrPort, listenerPort)
		upnpMap = err == nil
		log.WithFields(log.Fields{"module": logModule, "err": err}).Info("get UPNP external address")
	}

	// Get the IPv4 available
	if extAddr == nil {
		if ip, err := ExternalIPv4(); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("get ipv4 external address")
			log.Panic("get ipv4 external address fail!")
		} else {
			extAddr = NewNetAddressIPPort(net.ParseIP(ip), uint16(lAddrPort))
			log.WithFields(log.Fields{"module": logModule, "addr": extAddr}).Info("get ipv4 external address success")
		}
	}

	dl := &DefaultListener{
		listener:    listener,
		intAddr:     intAddr,
		extAddr:     extAddr,
		connections: make(chan net.Conn, numBufferedConnections),
	}
	dl.BaseService = *cmn.NewBaseService(nil, "DefaultListener", dl)
	dl.Start() // Started upon construction
	if upnpMap {
		return dl, true
	}

	conn, err := net.DialTimeout("tcp", extAddr.String(), 3*time.Second)
	if err != nil {
		return dl, false
	}
	conn.Close()
	return dl, true
}

//OnStart start listener
func (l *DefaultListener) OnStart() error {
	l.BaseService.OnStart()
	go l.listenRoutine()
	return nil
}

//OnStop stop listener
func (l *DefaultListener) OnStop() {
	l.BaseService.OnStop()
	l.listener.Close()
}

//listenRoutine Accept connections and pass on the channel
func (l *DefaultListener) listenRoutine() {
	for {
		conn, err := l.listener.Accept()
		if !l.IsRunning() {
			break // Go to cleanup
		}
		// listener wasn't stopped,
		// yet we encountered an error.
		if err != nil {
			log.Panic(err)
		}
		l.connections <- conn
	}
	// Cleanup
	close(l.connections)
}

//Connections a channel of inbound connections. It gets closed when the listener closes.
func (l *DefaultListener) Connections() <-chan net.Conn {
	return l.connections
}

//InternalAddress listener internal address
func (l *DefaultListener) InternalAddress() *NetAddress {
	return l.intAddr
}

//ExternalAddress listener external address for remote peer dial
func (l *DefaultListener) ExternalAddress() *NetAddress {
	return l.extAddr
}

// NetListener the returned listener is already Accept()'ing. So it's not suitable to pass into http.Serve().
func (l *DefaultListener) NetListener() net.Listener {
	return l.listener
}

//String string of default listener
func (l *DefaultListener) String() string {
	return fmt.Sprintf("Listener(@%v)", l.extAddr)
}
