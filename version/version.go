package version

import (
	"sync"

	gover "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/fatih/set.v0"
)

const (
	noUpdate uint16 = iota
	hasUpdate
	hasMUpdate
)

var (
	// The full version string
	Version = "1.0.5"
	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit string
	Status    *UpdateStatus
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
	Status = &UpdateStatus{
		maxVerSeen:    Version,
		notified:      false,
		seedSet:       set.New(),
		versionStatus: noUpdate,
	}
}

type UpdateStatus struct {
	sync.RWMutex
	maxVerSeen    string
	notified      bool
	seedSet       *set.Set
	versionStatus uint16
}

func (s *UpdateStatus) AddSeed(seedAddr string) {
	s.Lock()
	defer s.Unlock()
	s.seedSet.Add(seedAddr)
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
func (s *UpdateStatus) CheckUpdate(localVerStr string, remoteVerStr string, remoteAddr string) error {
	s.Lock()
	defer s.Unlock()

	if s.notified || !s.seedSet.Has(remoteAddr) {
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
		s.versionStatus = hasUpdate
		s.maxVerSeen = remoteVerStr
	}
	if remoteVersion.Segments()[0] > localVersion.Segments()[0] {
		s.versionStatus = hasMUpdate
	}
	if s.versionStatus != noUpdate {
		log.WithFields(log.Fields{
			"Current version": localVerStr,
			"Newer version":   remoteVerStr,
			"seed":            remoteAddr,
		}).Warn("Please update your bytomd via https://github.com/Bytom/bytom/releases/ or http://bytom.io/wallet/")
		s.notified = true
	}
	return nil
}

func (s *UpdateStatus) MaxVerSeen() string {
	s.RLock()
	defer s.RUnlock()
	return s.maxVerSeen
}

func (s *UpdateStatus) VersionStatus() uint16 {
	s.RLock()
	defer s.RUnlock()
	return s.versionStatus
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
