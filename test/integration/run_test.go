package integration

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
	"strings"
	"context"

	"github.com/bytom/blockchain"
	"github.com/bytom/blockchain/rpc"
	cfg "github.com/bytom/config"
	"github.com/bytom/node"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/bytom/env"
)

const (
	// Success indicates the rpc calling is successful.
	Success = iota
	// ErrLocalExe indicates error occurs before the rpc calling.
	ErrLocalExe
	// ErrConnect indicates error occurs connecting to the bytomd, e.g.,
	// bytomd can't parse the received arguments.
	ErrConnect
	// ErrLocalParse indicates error occurs locally when parsing the response.
	ErrLocalParse
	// ErrRemote indicates error occurs in bytomd.
	ErrRemote
)

var (
	home    = blockchain.HomeDirFromEnvironment()
	coreURL = env.String("BYTOM_URL", "http://localhost:9888")
)

func mockConfig() *cfg.Config {
	var config = cfg.DefaultConfig()
	config.Wallet.Enable = true
	config.Mining = true
	config.ApiAddress = "127.0.0.1:9888"
	return config
}

func testNet() bool {
	data, exitCode := clientCall("/net-info")
	if exitCode != Success {
		return false
	}
	dataMap, ok := data.(map[string]interface{})
	if ok && dataMap["listening"].(bool) && dataMap["syncing"].(bool) && dataMap["mining"].(bool) {
		return true
	}
	return false
}

func TestRunNode(t *testing.T) {
	// Create & start node
	config := mockConfig()
	n := node.NewNodeDefault(config)
	if _, err := n.Start(); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	go func() {
		time.Sleep(3000 * time.Millisecond)
		if testNet() {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}()
	// Trap signal, run forever.
	n.RunForever()
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func mustRPCClient() *rpc.Client {
	// TODO(kr): refactor some of this cert-loading logic into bytom/blockchain
	// and use it from cored as well.
	// Note that this function, unlike maybeUseTLS in cored,
	// does not load the cert and key from env vars,
	// only from the filesystem.
	certFile := filepath.Join(home, "tls.crt")
	keyFile := filepath.Join(home, "tls.key")
	config, err := blockchain.TLSConfig(certFile, keyFile, "")
	if err == blockchain.ErrNoTLS {
		return &rpc.Client{BaseURL: *coreURL}
	} else if err != nil {
		jww.ERROR.Println("loading TLS cert:", err)
	}

	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSClientConfig:       config,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	url := *coreURL
	if strings.HasPrefix(url, "http:") {
		url = "https:" + url[5:]
	}

	return &rpc.Client{
		BaseURL: url,
		Client:  &http.Client{Transport: t},
	}
}

func clientCall(path string, req ...interface{}) (interface{}, int) {

	var response = &blockchain.Response{}
	var request interface{}

	if req != nil {
		request = req[0]
	}

	client := mustRPCClient()
	client.Call(context.Background(), path, request, response)

	switch response.Status {
	case blockchain.FAIL:
		jww.ERROR.Println(response.Msg)
		return nil, ErrRemote
	case "":
		jww.ERROR.Println("Unable to connect to the bytomd")
		return nil, ErrConnect
	}

	return response.Data, Success
}
