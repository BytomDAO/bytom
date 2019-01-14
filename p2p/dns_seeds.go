package p2p

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
)

const (
	logModule = "p2p"

	mainnetPort = "46657"
	testnetPort = "46656"
)

var (
	errChainID    = errors.New("unsupported chain id")
	errInvalidIP  = errors.New("invalid ip address")
	errDNSTimeout = errors.New("get dns seed timeout")
)

var (
	mainnetSeeds = []string{"www.mainnetseed.yahtoo.fun"}
	testnetSeeds = []string{"www.testnetseed.yahtoo.fun"}

	dnsTimeout = 5 * time.Second
)

// QueryDNSSeeds Query the DNS seeds.
func QueryDNSSeeds(chainID string, lookupHost func(host string) (addrs []string, err error)) ([]string, error) {
	var dnsSeeds []string
	var port string

	switch chainID {
	case "mainnet":
		dnsSeeds = mainnetSeeds
		port = mainnetPort
	case "wisdom":
		dnsSeeds = testnetSeeds
		port = testnetPort
	case "solonet":
		return nil, nil
	default:
		return nil, errChainID
	}

	resultCh := make(chan *[]string, 1)
	for _, dnsSeed := range dnsSeeds {
		go queryDNSSeeds(lookupHost, resultCh, dnsSeed, port)
	}

	for {
		select {
		case result := <-resultCh:
			return *result, nil
		case <-time.After(dnsTimeout):
			return nil, errDNSTimeout
		}
	}
}

func queryDNSSeeds(lookupHost func(host string) (addrs []string, err error), resultCh chan *[]string, dnsSeed, port string) {
	var seeds []string

	//TODO add proxy
	addrs, err := lookupHost(dnsSeed)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err, "dnsSeed": dnsSeed}).Error("fail on look up host")
		return
	}

	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip == nil {
			log.WithFields(log.Fields{"module": logModule, "err": errInvalidIP, "dnsSeed": dnsSeed}).Error("fail on parse IP")
			return
		}

		seeds = append(seeds, net.JoinHostPort(addr, port))
	}

	resultCh <- &seeds
}
