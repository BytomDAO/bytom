package blockchain

import (
	"os"
	"path/filepath"
)

// HomeDirFromEnvironment returns the directory to use
// for reading config and storing variable data.
// It returns $BYTOM_HOME,
// or, if that is empty, $HOME/.chaincore.
func HomeDirFromEnvironment() string {
	if s := os.Getenv("BYTOM_HOME"); s != "" {
		return s
	}
	return filepath.Join(os.Getenv("HOME"), ".bytom")
}
