package version

import (
	gover "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/fatih/set.v0"
)

var (
	// The full version string
	Version = "1.0.4"
	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit string
	Update    uint16
	notified  bool
	SeedSet   = set.New()
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
}

func CheckUpdate(localVerStr string, remoteVerStr string, remoteAddr string) error {
	if !SeedSet.Has(remoteAddr) || notified {
		return nil
	}

	localVersion, err := gover.NewVersion(localVerStr)
	if err != nil {
		return err
	}
	remoteVersion, err := gover.NewVersion(remoteVerStr)
	if err != nil {
		return err
	}
	if remoteVersion.GreaterThan(localVersion) {
		Update++
	}
	if remoteVersion.Segments()[0] > localVersion.Segments()[0] {
		Update++
	}
	if Update > 0 {
		log.WithFields(log.Fields{
			"Current version": localVerStr,
			"Newer version":   remoteVerStr,
			"seed":            remoteAddr}).
			Warn("Please update your bytomd via https://github.com/Bytom/bytom/releases/ or http://bytom.io/wallet/")
	}
	notified = true
	return nil
}

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
