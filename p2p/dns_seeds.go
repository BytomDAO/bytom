package p2p

import (
	"net"

	"github.com/bytom/errors"
)

const (
	mainnetPort = "46657"
	testnetPort = "46656"
)

var (
	mainnetSeeds = []string{"www.mainnetseed.yahtoo.fun"}
	testnetSeeds = []string{"www.testnetseed.yahtoo.fun"}
)

var (
	errChainID   = errors.New("unsupported chain id")
	errInvalidIP = errors.New("invalid ip address")
)

// QueryDNSSeeds Query the DNS seeds.
func QueryDNSSeeds(chainID string, lookupHost func(host string) (addrs []string, err error)) ([]string, error) {
	var seeds []string
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

	for _, seed := range dnsSeeds {
		//TODO add proxy
		addrs, err := lookupHost(seed)
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			if ip := net.ParseIP(addr); ip == nil {
				return nil, errInvalidIP
			}

			seeds = append(seeds, net.JoinHostPort(addr, port))
		}
	}

	return seeds, nil
}
