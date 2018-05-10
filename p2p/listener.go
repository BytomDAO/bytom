package p2p

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/bytom/p2p/upnp"
	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"
	tlog "github.com/tendermint/tmlibs/log"
)

type Listener interface {
	Connections() <-chan net.Conn
	InternalAddress() *NetAddress
	ExternalAddress() *NetAddress
	String() string
	Stop() bool
}

// Implements Listener
type DefaultListener struct {
	cmn.BaseService

	listener    net.Listener
	intAddr     *NetAddress
	extAddr     *NetAddress
	connections chan net.Conn
}

const (
	numBufferedConnections = 10
	defaultExternalPort    = 8770
	tryListenSeconds       = 5
)

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

// skipUPNP: If true, does not try getUPNPExternalAddress()
func NewDefaultListener(protocol string, lAddr string, skipUPNP bool, logger tlog.Logger) (Listener, bool) {
	// Local listen IP & port
	lAddrIP, lAddrPort := splitHostPort(lAddr)

	// Create listener
	var listener net.Listener
	var err error
	var getExtIP = false
	var listenerStatus = false

	for i := 0; i < tryListenSeconds; i++ {
		listener, err = net.Listen(protocol, lAddr)
		if err == nil {
			break
		} else if i < tryListenSeconds-1 {
			time.Sleep(time.Second * 1)
		}
	}
	if err != nil {
		cmn.PanicCrisis(err)
	}
	// Actual listener local IP & port
	listenerIP, listenerPort := splitHostPort(listener.Addr().String())
	log.WithFields(log.Fields{
		"ip":   listenerIP,
		"port": listenerPort,
	}).Info("Local listener")

	// Determine internal address...
	var intAddr *NetAddress
	intAddr, err = NewNetAddressString(lAddr)
	if err != nil {
		cmn.PanicCrisis(err)
	}

	// Determine external address...
	var extAddr *NetAddress
	if !skipUPNP {
		// If the lAddrIP is INADDR_ANY, try UPnP
		if lAddrIP == "" || lAddrIP == "0.0.0.0" {
			extAddr = getUPNPExternalAddress(lAddrPort, listenerPort)
			if extAddr != nil {
				getExtIP = true
				listenerStatus = true
			}
		}
	}
	if extAddr == nil {
		if address := GetIP(); address.Success == true {
			extAddr = NewNetAddressIPPort(net.ParseIP(address.Ip), uint16(lAddrPort))
			getExtIP = true
		}
	}
	// Otherwise just use the local address...
	if extAddr == nil {
		extAddr = getNaiveExternalAddress(listenerPort)
	}
	if extAddr == nil {
		cmn.PanicCrisis("Could not determine external address!")
	}

	dl := &DefaultListener{
		listener:    listener,
		intAddr:     intAddr,
		extAddr:     extAddr,
		connections: make(chan net.Conn, numBufferedConnections),
	}
	dl.BaseService = *cmn.NewBaseService(logger, "DefaultListener", dl)
	dl.Start() // Started upon construction

	if !listenerStatus && getExtIP {
		conn, err := net.DialTimeout("tcp", extAddr.String(), 3*time.Second)

		if err != nil && conn == nil {
			log.Error("Could not open listen port")
		}

		if err == nil && conn != nil {
			log.Info("Success open listen port")
			listenerStatus = true
			conn.Close()
		}
	}

	return dl, listenerStatus
}

func (l *DefaultListener) OnStart() error {
	l.BaseService.OnStart()
	go l.listenRoutine()
	return nil
}

func (l *DefaultListener) OnStop() {
	l.BaseService.OnStop()
	l.listener.Close()
}

// Accept connections and pass on the channel
func (l *DefaultListener) listenRoutine() {
	for {
		conn, err := l.listener.Accept()

		if !l.IsRunning() {
			break // Go to cleanup
		}

		// listener wasn't stopped,
		// yet we encountered an error.
		if err != nil {
			cmn.PanicCrisis(err)
		}

		l.connections <- conn
	}

	// Cleanup
	close(l.connections)
	for _ = range l.connections {
		// Drain
	}
}

// A channel of inbound connections.
// It gets closed when the listener closes.
func (l *DefaultListener) Connections() <-chan net.Conn {
	return l.connections
}

func (l *DefaultListener) InternalAddress() *NetAddress {
	return l.intAddr
}

func (l *DefaultListener) ExternalAddress() *NetAddress {
	return l.extAddr
}

// NOTE: The returned listener is already Accept()'ing.
// So it's not suitable to pass into http.Serve().
func (l *DefaultListener) NetListener() net.Listener {
	return l.listener
}

func (l *DefaultListener) String() string {
	return fmt.Sprintf("Listener(@%v)", l.extAddr)
}

/* external address helpers */

// UPNP external address discovery & port mapping
func getUPNPExternalAddress(externalPort, internalPort int) *NetAddress {
	log.Info("Getting UPNP external address")
	nat, err := upnp.Discover()
	if err != nil {
		log.WithField("error", err).Error("Could not perform UPNP discover")
		return nil
	}

	ext, err := nat.GetExternalAddress()
	if err != nil {
		log.WithField("error", err).Error("Could not perform UPNP external address")
		return nil
	}

	// UPnP can't seem to get the external port, so let's just be explicit.
	if externalPort == 0 {
		externalPort = defaultExternalPort
	}

	externalPort, err = nat.AddPortMapping("tcp", externalPort, internalPort, "bytomd", 0)
	if err != nil {
		log.WithField("error", err).Error("Could not add UPNP port mapping")
		return nil
	}

	log.WithField("address", ext).Info("Got UPNP external address")
	return NewNetAddressIPPort(ext, uint16(externalPort))
}

// TODO: use syscalls: http://pastebin.com/9exZG4rh
func getNaiveExternalAddress(port int) *NetAddress {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		cmn.PanicCrisis(cmn.Fmt("Could not fetch interface addresses: %v", err))
	}

	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		v4 := ipnet.IP.To4()
		if v4 == nil || v4[0] == 127 {
			continue
		} // loopback
		return NewNetAddressIPPort(ipnet.IP, uint16(port))
	}
	return nil
}
