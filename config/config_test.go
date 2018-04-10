package config

import (
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
