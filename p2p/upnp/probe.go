package upnp

import (
	"errors"
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

type UPNPCapabilities struct {
	PortMapping bool
	Hairpin     bool
}

func makeUPNPListener(intPort int, extPort int) (NAT, net.Listener, net.IP, error) {
	nat, err := Discover()
	if err != nil {
		return nil, nil, nil, errors.New(fmt.Sprintf("NAT upnp could not be discovered: %v", err))
	}
	log.WithField("ourIP", nat.(*upnpNAT).ourIP).Info("outIP:")

	ext, err := nat.GetExternalAddress()
	if err != nil {
		return nat, nil, nil, errors.New(fmt.Sprintf("External address error: %v", err))
	}
	log.WithField("address", ext).Info("External address")

	port, err := nat.AddPortMapping("tcp", extPort, intPort, "Tendermint UPnP Probe", 0)
	if err != nil {
		return nat, nil, ext, errors.New(fmt.Sprintf("Port mapping error: %v", err))
	}
	log.WithField("port", port).Info("Port mapping mapped")

	// also run the listener, open for all remote addresses.
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", intPort))
	if err != nil {
		return nat, nil, ext, errors.New(fmt.Sprintf("Error establishing listener: %v", err))
	}
	return nat, listener, ext, nil
}

func testHairpin(listener net.Listener, extAddr string) (supportsHairpin bool) {
	// Listener
	go func() {
		inConn, err := listener.Accept()
		if err != nil {
			log.WithField("error", err).Error("Listener.Accept() error")
			return
		}
		log.WithFields(log.Fields{
			"LocalAddr":  inConn.LocalAddr(),
			"RemoteAddr": inConn.RemoteAddr(),
		}).Info("Accepted incoming connection")
		buf := make([]byte, 1024)
		n, err := inConn.Read(buf)
		if err != nil {
			log.WithField("error", err).Error("Incoming connection read error")
			return
		}
		log.Infof("Incoming connection read %v bytes: %X", n, buf)
		if string(buf) == "test data" {
			supportsHairpin = true
			return
		}
	}()

	// Establish outgoing
	outConn, err := net.Dial("tcp", extAddr)
	if err != nil {
		log.WithField("error", err).Error("Outgoing connection dial error")
		return
	}

	n, err := outConn.Write([]byte("test data"))
	if err != nil {
		log.WithField("error", err).Error("Outgoing connection write error")
		return
	}
	log.Infof("Outgoing connection wrote %v bytes", n)

	// Wait for data receipt
	time.Sleep(1 * time.Second)
	return
}

func Probe() (caps UPNPCapabilities, err error) {
	log.Info("Probing for UPnP!")

	intPort, extPort := 8001, 8001

	nat, listener, ext, err := makeUPNPListener(intPort, extPort)
	if err != nil {
		return
	}
	caps.PortMapping = true

	// Deferred cleanup
	defer func() {
		err = nat.DeletePortMapping("tcp", intPort, extPort)
		if err != nil {
			log.WithField("error", err).Error("Port mapping delete error")
		}
		listener.Close()
	}()

	supportsHairpin := testHairpin(listener, fmt.Sprintf("%v:%v", ext, extPort))
	if supportsHairpin {
		caps.Hairpin = true
	}

	return
}
