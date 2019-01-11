package p2p

import (
	"github.com/bytom/errors"
	"net"
)

const (
	mainnetPort = "46657"
	testnetPort = "46656"
)

var mainnetSeeds = []string{"www.mainnetseed.yahtoo.fun"}
var testnetSeeds = []string{"www.testnetseed.yahtoo.fun"}

var errChainID = errors.New("unsupported chain id")

// QueryDNSSeeds Query the DNS seeds.
func QueryDNSSeeds(chainID string) ([]string, error) {
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
		addrs, err := net.LookupHost(seed)
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			seeds = append(seeds, net.JoinHostPort(addr, port))
		}
	}

	return seeds, nil
}
