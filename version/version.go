package version

import (
	gover "github.com/hashicorp/go-version"
	"gopkg.in/fatih/set.v0"

	"github.com/bytom/p2p"
)

var (
	// The full version string
	Version = "1.0.4"
	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit string
	Update    bool
	SUpdate   bool
	SeedSet   *set.Set
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
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

func CheckUpdate(local *NodeInfo, remote *NodeInfo) {
	if SeedSet.Has(remote.PubKey) {
		localVersion, _ := gover.NewVersion(local.Version)
		remoteVersion, _ := gover.NewVersion(remote.Version)

		if remoteVersion.GreaterThan(localVersion) {
			Update = true
			log.Info("[Current version is out-dated.] " +
				"Current version: " + local.Version +
				". Newer version " + remote.Version + " seen from seed " + remote.RemoteAddr +
				". Please update your bytomd via " +
				"https://github.com/Bytom/bytom/releases or http://bytom.io/wallet/.")
		}

		if remoteVersion.Segments()[0] > localVersion.Segments()[0] {
			SUpdate = true
			log.Info("[Current version is too old.] " +
				"Current version: " + local.Version +
				". Newer version " + remote.Version + " seen from seed " + remote.RemoteAddr +
				". Please update your bytomd via " +
				"https://github.com/Bytom/bytom/releases or http://bytom.io/wallet/.")
		}
	}
}
