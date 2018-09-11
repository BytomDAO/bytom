package version

import (
	"testing"

	gover "github.com/hashicorp/go-version"
)

func TestCompare(t *testing.T) {
	v1, err := gover.NewVersion(Version)
	if err != nil {
		t.Fatal("Version 1 format error.")
	}
	v2, err := gover.NewVersion(Version + "+f873dfca")
	if err != nil {
		t.Fatal("Version 2 format error.")
	}
	if v1.GreaterThan(v2) || v1.GreaterThan(v2) {
		t.Error("Version comparison error.")
	}
}

func TestCompatibleWith(t *testing.T) {
	cases := []struct {
		a      string
		b      string
		result bool
	}{
		{
			"1.0.4",
			"1.0.4",
			true,
		},
		{
			"1.0.4",
			"1.0.5",
			true,
		},
		{
			"1.0.4",
			"1.1.5",
			true,
		},
		{
			"1.0.5",
			"1.0.5-90825109",
			true,
		},
		{
			"1.0.5",
			"1.0.5+90825109",
			true,
		},
		{
			"1.0.5",
			"2.0.5",
			false,
		},
		{
			"1.0.5-90825109",
			"1.0.5+90825109",
			true,
		},
	}

	for i, c := range cases {
		Version = c.a
		if result, _ := CompatibleWith(c.b); c.result != result {
			t.Errorf("case %d: got %t want %t", i, c.result, result)
		}
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
