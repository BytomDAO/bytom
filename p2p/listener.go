package p2p

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/bytom/p2p/upnp"
	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"
)

//Listener subset of the methods of DefaultListener
type Listener interface {
	Connections() <-chan net.Conn
	InternalAddress() *NetAddress
	ExternalAddress() *NetAddress
	String() string
	Stop() bool
}

//DefaultListener Implements bytomd server Listener
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

//NewDefaultListener create a default listener
func NewDefaultListener(protocol string, lAddr string, skipUPNP bool) (Listener, bool) {
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
	log.Info("Local listener", " ip:", listenerIP, " port:", listenerPort)

	// Determine internal address...
	var intAddr *NetAddress
	intAddr, err = NewNetAddressString(lAddr)
	if err != nil {
		cmn.PanicCrisis(err)
	}

	// Determine external address...
	var extAddr *NetAddress
	//skipUPNP: If true, does not try getUPNPExternalAddress()
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
		extAddr = getNaiveExternalAddress(listenerPort, false)
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
	dl.BaseService = *cmn.NewBaseService(nil, "DefaultListener", dl)
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

//getUPNPExternalAddress UPNP external address discovery & port mapping
func getUPNPExternalAddress(externalPort, internalPort int) *NetAddress {
	log.Info("Getting UPNP external address")
	nat, err := upnp.Discover()
	if err != nil {
		log.Info("Could not perform UPNP discover. error:", err)
		return nil
	}

	ext, err := nat.GetExternalAddress()
	if err != nil {
		log.Info("Could not perform UPNP external address. error:", err)
		return nil
	}

	// UPnP can't seem to get the external port, so let's just be explicit.
	if externalPort == 0 {
		externalPort = defaultExternalPort
	}

	externalPort, err = nat.AddPortMapping("tcp", externalPort, internalPort, "bytomd", 0)
	if err != nil {
		log.Info("Could not add UPNP port mapping. error:", err)
		return nil
	}

	log.Info("Got UPNP external address ", ext)
	return NewNetAddressIPPort(ext, uint16(externalPort))
}

func getNaiveExternalAddress(port int, settleForLocal bool) *NetAddress {
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
		if v4 == nil || (!settleForLocal && v4[0] == 127) {
			continue
		} // loopback
		return NewNetAddressIPPort(ipnet.IP, uint16(port))
	}

	// try again, but settle for local
	log.Info("Node may not be connected to internet. Settling for local address")
	return getNaiveExternalAddress(port, true)
}
