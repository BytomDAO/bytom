package dht

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
)

var (
	errInvalidIP     = errors.New("invalid ip address")
	errDNSTimeout    = errors.New("get dns seed timeout")
	errDNSSeedsEmpty = errors.New("dns seeds is empty")

	dnsTimeout = 5 * time.Second
)

// QueryDNSSeeds Query the DNS seeds.
func QueryDNSSeeds(lookupHost func(host string) (addrs []string, err error)) ([]string, error) {
	if len(consensus.ActiveNetParams.DNSSeeds) == 0 {
		return nil, errDNSSeedsEmpty
	}

	resultCh := make(chan *[]string, 1)
	for _, dnsSeed := range consensus.ActiveNetParams.DNSSeeds {
		go queryDNSSeeds(lookupHost, resultCh, dnsSeed, consensus.ActiveNetParams.DefaultPort)
	}

	select {
	case result := <-resultCh:
		return *result, nil
	case <-time.After(dnsTimeout):
		return nil, errDNSTimeout
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
	if len(seeds) == 0 {
		return
	}
	//if channel is full, drop it
	select {
	case resultCh <- &seeds:
	default:
	}
}
