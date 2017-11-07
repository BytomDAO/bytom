package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ensureFiles(t *testing.T, rootDir string, files ...string) {
	for _, f := range files {
		p := rootify(rootDir, f)
		_, err := os.Stat(p)
		assert.Nil(t, err, p)
	}
}

func TestEnsureRoot(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	// setup temp dir for test
	tmpDir, err := ioutil.TempDir("", "config-test")
	require.Nil(err)
	defer os.RemoveAll(tmpDir)

	// create root dir
	EnsureRoot(tmpDir, "mainnet")

	// make sure config is set properly
	data, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
	require.Nil(err)
	assert.Equal([]byte(selectNetwork("mainnet")), data)

	ensureFiles(t, tmpDir, "data")
}
