package version

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	gover "github.com/hashicorp/go-version"
)

func TestCompare(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	i := rand.Int63n(0xffffffff)
	verb := fmt.Sprintf("%%%dx", revLen)
	rev := fmt.Sprintf(verb, i)

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
