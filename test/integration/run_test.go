package integration

import (
	"fmt"

	cfg "github.com/bytom/bytom/config"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/util"
)

// Mock config.
func mockConfig() *cfg.Config {
	var config = cfg.DefaultConfig()
	config.Wallet.Disable = false
	config.Mining = true
	config.ApiAddress = "127.0.0.1:9888"
	return config
}

// Test net-info call api.
func testNet() bool {
	data, exitCode := util.ClientCall("/net-info")
	if exitCode != util.Success {
		return false
	}
	dataMap, ok := data.(map[string]interface{})
	if ok && dataMap["listening"].(bool) && dataMap["syncing"].(bool) && dataMap["mining"].(bool) {
		return true
	}
	return false
}

// Test create-key delete-key list-key api and function.
func testKey() bool {
	var key = struct {
		Alias    string `json:"alias"`
		Password string `json:"password"`
	}{Alias: "alice", Password: "123456"}

	data, exitCode := util.ClientCall("/create-key", &key)
	if exitCode != util.Success {
		return false
	}
	dataMap, ok := data.(map[string]interface{})
	if (ok && dataMap["alias"].(string) == "alice") == false {
		return false
	}

	_, exitCode1 := util.ClientCall("/list-keys")
	if exitCode1 != util.Success {
		return false
	}

	fmt.Println("dataMap", dataMap)
	xpub := new(chainkd.XPub)
	if err := xpub.UnmarshalText([]byte(dataMap["xpub"].(string))); err != nil {
		return false
	}

	var key1 = struct {
		Password string
		XPub     chainkd.XPub `json:"xpubs"`
	}{XPub: *xpub, Password: "123456"}

	if _, exitCode := util.ClientCall("/delete-key", &key1); exitCode != util.Success {
		return false
	}

	return true
}

// Test node running.
/*func TestRunNode(t *testing.T) {
	// Create & start node
	config := mockConfig()
	n := node.NewNodeDefault(config)
	if _, err := n.Start(); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	go func() {
		time.Sleep(3000 * time.Millisecond)
		if testNet() && testKey() {
			os.RemoveAll("./data")
			os.RemoveAll("./keystore")
			os.Exit(0)
		} else {
			os.RemoveAll("./data")
			os.RemoveAll("./keystore")
			os.Exit(1)
		}
	}()
	// Trap signal, run forever.
	n.RunForever()
}*/
