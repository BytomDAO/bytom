package version

import (
	"fmt"
	"testing"

	gover "github.com/hashicorp/go-version"
	"gopkg.in/fatih/set.v0"
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

func TestCheckUpdate(t *testing.T) {
	cases := []struct {
		desc           string
		localVer       string
		remotePeers    []string
		wantStatus     uint16
		wantmaxVerSeen string
		wantNotified   bool
	}{
		{
			desc:           "has large version number update",
			localVer:       "1.0",
			remotePeers:    []string{"1.0", "2.0", "1.0.3"},
			wantStatus:     hasMUpdate,
			wantmaxVerSeen: "2.0",
			wantNotified:   true,
		},
		{
			desc:           "some remote version less than local version, but some remote verison larger than local version",
			localVer:       "1.0",
			remotePeers:    []string{"0.8", "1.1", "1.0.3", "0.9"},
			wantStatus:     hasUpdate,
			wantmaxVerSeen: "1.1",
			wantNotified:   true,
		},
		{
			desc:           "has small version number update",
			localVer:       "1.0",
			remotePeers:    []string{"1.0", "1.0.3", "1.0.2"},
			wantStatus:     hasUpdate,
			wantmaxVerSeen: "1.0.3",
			wantNotified:   true,
		},
		{
			desc:           "the remote equals to local version",
			localVer:       "1.0",
			remotePeers:    []string{"1.0", "1.0", "1.0"},
			wantStatus:     noUpdate,
			wantmaxVerSeen: "1.0",
			wantNotified:   false,
		},
		{
			desc:           "the remote version less than local version",
			localVer:       "1.0",
			remotePeers:    []string{"0.8", "0.8", "0.8"},
			wantStatus:     noUpdate,
			wantmaxVerSeen: "1.0",
			wantNotified:   false,
		},
	}

	for i, c := range cases {
		status := &UpdateStatus{
			maxVerSeen:    c.localVer,
			notified:      false,
			seedSet:       set.New(),
			versionStatus: noUpdate,
		}
		for i, remoteVer := range c.remotePeers {
			peer := fmt.Sprintf("peer%d", i)
			status.seedSet.Add(peer)
			if err := status.CheckUpdate(c.localVer, remoteVer, peer); err != nil {
				t.Fatal(err)
			}
		}

		if status.versionStatus != c.wantStatus {
			t.Errorf("#%d(%s) got version status:%d, want version status:%d", i, c.desc, status.versionStatus, c.wantStatus)
		}

		if status.notified != c.wantNotified {
			t.Errorf("#%d(%s) got notified:%t, want notified:%t", i, c.desc, status.notified, c.wantNotified)
		}

		if status.maxVerSeen != c.wantmaxVerSeen {
			t.Errorf("#%d(%s) got max version seen%s, want max version seen%s", i, c.desc, status.maxVerSeen, c.wantmaxVerSeen)
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
