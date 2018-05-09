package p2p

import (
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

var ipCheckServices = []string{
	"http://members.3322.org/dyndns/getip",
	"http://ifconfig.me/",
	"http://icanhazip.com/",
	"http://ifconfig.io/ip",
	"http://ident.me/",
	"http://whatismyip.akamai.com/",
	"http://myip.dnsomatic.com/",
	"http://diagnostic.opendns.com/myip",
	"http://myexternalip.com/raw",
}

type IpResult struct {
	Success bool
	Ip      string
}

var timeout = time.Duration(10)

func GetIP() *IpResult {
	resultCh := make(chan *IpResult)
	for _, s := range ipCheckServices {
		go ipAddress(s, resultCh)
	}

	for {
		select {
		case result := <-resultCh:
			if result.Success {
				return result
			}
		case <-time.After(time.Second * timeout):
			return &IpResult{false, ""}
		}
	}
}

func ipAddress(service string, done chan<- *IpResult) {
	client := http.Client{Timeout: time.Duration(timeout * time.Second)}
	resp, err := client.Get(service)
	if err != nil {
		sendResult(&IpResult{false, ""}, done)
		return
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		sendResult(&IpResult{false, ""}, done)
		return
	}

	address := strings.TrimSpace(string(data))
	if net.ParseIP(address) != nil {
		sendResult(&IpResult{true, address}, done)
		return
	}

	sendResult(&IpResult{false, ""}, done)
}

func sendResult(result *IpResult, done chan<- *IpResult) {
	select {
	case done <- result:
		return
	default:
		return
	}
}
