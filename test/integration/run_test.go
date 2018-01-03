package integration

import (
	"os"
	"testing"
	"time"

	cfg "github.com/bytom/config"
	"github.com/bytom/node"
)

func TestRunNode(t *testing.T) {
	var config = cfg.DefaultConfig()
	// Create & start node
	n := node.NewNodeDefault(config)
	if _, err := n.Start(); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	go func() {
		time.Sleep(3000 * time.Millisecond)
		os.Exit(0)
	}()
	// Trap signal, run forever.
	n.RunForever()
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
