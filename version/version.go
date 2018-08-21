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

func CheckUpdate(localVerStr string, remoteVerStr string, remoteAddr string) {
	if !SeedSet.Has(remoteAddr) || notified {
		return
	}

	localVersion, _ := gover.NewVersion(localVerStr)
	remoteVersion, _ := gover.NewVersion(remoteVerStr)
	if remoteVersion.GreaterThan(localVersion) {
		Update++
	}
	if remoteVersion.Segments()[0] > localVersion.Segments()[0] {
		Update++
	}
	if Update > 0 {
		log.Info("Current version: " + localVerStr +
			". Newer version: " + remoteVerStr + " seen from seed: " + remoteAddr +
			". Please update your bytomd via " +
			"https://github.com/Bytom/bytom/releases/ or http://bytom.io/wallet/.")
	}
	notified = true
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
