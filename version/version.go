package version

import (
	gover "github.com/hashicorp/go-version"
)

const (
	deprecateBelow = "1.0.0"
	notifyLimit    = uint16(3)
)

var (
	// The full version string
	Version = "1.0.4"
	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit     string
	notifiedTimes = uint16(0)
	maxVerSeen    *gover.Version
	SUpdate       bool
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
	maxVerSeen, _ = gover.NewVersion(Version)
}

/* 						Functions for version-control					*/
// -------------------------------start----------------------------------

// CompatibleWith checks whether the remote peer version is compatible with the
// node itself.
// RULES:
// | local |           remote           |
// |   -   |             -              |
// | 1.0.3 | same major&moinor version. |
// | 1.0.4 |     same major version.    |
func CompatibleWith(remoteVerStr string) (bool, error) {
	localVersion, err := gover.NewVersion(Version)
	if err != nil {
		return false, err
	}
	remoteVersion, err := gover.NewVersion(remoteVerStr)
	if err != nil {
		return false, err
	}
	return (localVersion.Segments()[0] == remoteVersion.Segments()[0]), nil
}

// OlderThan checks whether the node version is older than a remote peer.
// remoteVer is supposed to always be corresponding to a seed
func OlderThan(remoteVerStr string) (bool, error) {
	localVersion, err := gover.NewVersion(Version)
	if err != nil {
		return false, err
	}
	remoteVersion, err := gover.NewVersion(remoteVerStr)
	if err != nil {
		return false, err
	}

	// Reset notifiedTimes
	if remoteVersion.GreaterThan(maxVerSeen) {
		maxVerSeen = remoteVersion
		notifiedTimes = uint16(0)
	}

	return localVersion.LessThan(remoteVersion), nil
}

/* 						Functions for version-control					*/
// --------------------------------end-----------------------------------

// ShouldNotify tells whether bytomd or dashboard should notify the bytomd
// version is out of date.
func ShouldNotify(end string) bool {
	switch end {
	case "bytomd":
		notifiedTimes++
		return notifiedTimes <= notifyLimit
	case "dashboard":
		return notifiedTimes > 0
	default:
		return false
	}
}
