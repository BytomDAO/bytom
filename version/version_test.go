package version

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	gover "github.com/hashicorp/go-version"
)

func TestRevisionLen(t *testing.T) {
	if revisionLen > 16 {
		t.Error("revisionLen too long")
	}
}

func TestCompare(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	i := rand.Uint64()
	rev := fmt.Sprintf("%16x", i)[:revisionLen]

	v1, err := gover.NewVersion(Version)
	if err != nil {
		t.Error("Version 1 format error.")
	}
	v2, err := gover.NewVersion(Version + "+" + rev)
	if err != nil {
		t.Error("Version 2 format error.")
	}
	if v1.GreaterThan(v2) || v1.GreaterThan(v2) {
		t.Error("Version comparison error.")
	}
}

// In case someone edit the iota part and have the mapping changed:
// noUpdate: 0
// hasUpdate: 1
// hasMUpdate: 2
func TestFlag(t *testing.T) {
	if noUpdate != 0 {
		t.Error("noUpdate value error")
	}
	if hasUpdate != 1 {
		t.Error("hasUpdate value error")
	}
	if hasMUpdate != 2 {
		t.Error("noUpdate value error")
	}
}
