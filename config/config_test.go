package config

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	assert := assert.New(t)

	// set up some defaults
	cfg := DefaultConfig()
	assert.NotNil(cfg.P2P)

	// check the root dir stuff...
	cfg.SetRoot("/foo")
	cfg.DBPath = "/opt/data"

	assert.Equal("/opt/data", cfg.DBDir())

}

func TestNodeKey(t *testing.T) {
	tmpDir, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatalf("failed to create temporary data folder: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	config := DefaultConfig()
	config.BaseConfig.RootDir = tmpDir

	config.P2P.PrivateKey = "0fcbd0be11e35c35c41c686b7ca597bbcf8ecb78e320d01a93349c8ce9420ea4f26d0fbe651bb2c248d6727801329b589ed19e384c9e906d1da4ab2360558bc0"
	privKey, err := config.NodeKey()
	if err != nil {
		t.Fatal("test node key error:", err)
	}

	if strings.Compare(privKey, config.P2P.PrivateKey) != 0 {
		t.Fatal("test node key error. want:", config.P2P.PrivateKey, "got:", privKey)
	}

	config.P2P.PrivateKey = ""
	writePrivKey, err := config.NodeKey()
	if err != nil {
		t.Fatal("test node key error:", err)
	}

	readPrivKey, err := config.NodeKey()
	if err != nil {
		t.Fatal("test node key error:", err)
	}

	if strings.Compare(writePrivKey, readPrivKey) != 0 {
		t.Fatal("test node key error. write:", writePrivKey, "read:", readPrivKey)
	}
}
