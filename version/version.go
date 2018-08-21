package version

import (
	gover "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/fatih/set.v0"
	"sync"
)

const (
	noUpdate   uint16 = iota
	hasUpdate  uint16 = iota
	hasMUpdate uint16 = iota
)

var (
	// The full version string
	Version = "1.0.3"
	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit string
	SeedSet   = set.New()
	Status    *State
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
	Status = &State{
		notified:      false,
		VersionStatus: noUpdate,
	}
}

type State struct {
	sync.RWMutex
	notified      bool
	VersionStatus uint16
}

func (s *State) GetVersionStatus() uint16 {
	s.RLock()
	defer s.RUnlock()
	return s.VersionStatus
}

// CheckUpdate checks whether there is a newer version to update.
// If there is, it set the "Status" variable to a proper value.
// 	params:
// 		localVerStr: the version of the node itself
// 		remoteVerStr: the version received from a seed node.
// 		remoteAddr: the version received from a seed node.
// current rule:
// 		1. small update: seed version is higher than the node itself
// 		2. significant update: seed mojor version is higher than the node itself
func CheckUpdate(localVerStr string, remoteVerStr string, remoteAddr string) error {
	Status.Lock()
	defer Status.Unlock()

	if Status.notified || !SeedSet.Has(remoteAddr) {
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
		Status.VersionStatus = hasUpdate
	}
	if remoteVersion.Segments()[0] > localVersion.Segments()[0] {
		Status.VersionStatus = hasMUpdate
	}
	if Status.VersionStatus != noUpdate {
		log.WithFields(log.Fields{
			"Current version": localVerStr,
			"Newer version":   remoteVerStr,
			"seed":            remoteAddr}).
			Warn("Please update your bytomd via https://github.com/Bytom/bytom/releases/ or http://bytom.io/wallet/")
		Status.notified = true
	}
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
