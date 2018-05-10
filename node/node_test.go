package node

import (
	"io/ioutil"
	"os"
	"testing"

	cfg "github.com/bytom/config"
)

func TestNodeUsedDataDir(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary data directory: %v", err)
	}
	defer os.RemoveAll(dir)
	var config cfg.Config
	config.RootDir = dir
	if err := lockDataDirectory(&config); err != nil {
		t.Fatalf("Error: %v", err)
	}

	if err := lockDataDirectory(&config); err == nil {
		t.Fatalf("duplicate datadir failure mismatch: want %v", err)
	}
}
