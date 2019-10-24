// package version provide the version info for the node, and also provide
// support for version compatibility check and update notification.
//
// The version format should follow Semantic Versioning (https://semver.org/):
// MAJOR.MINOR.PATCH
//  1. MAJOR version when you make incompatible API changes,
//  2. MINOR version when you add functionality in a backwards-compatible manner, and
// 	3. PATCH version when you make backwards-compatible bug fixes.
//
// A pre-release version MAY be denoted by appending a hyphen and a series of
// dot separated identifiers immediately following the patch version.
// Examples:
// 1.0.0-alpha, 1.0.0-alpha.1, 1.0.0-0.3.7, 1.0.0-x.7.z.92.
// Precedence:
// 1. Pre-release versions have a lower precedence than the associated normal version!
//    Numeric identifiers always have lower precedence than non-numeric identifiers.
// 2. A larger set of pre-release fields has a higher precedence than a smaller set,
//    if all of the preceding identifiers are equal.
// 3. Example:
//    1.0.0-alpha < 1.0.0-alpha.1 < 1.0.0-alpha.beta < 1.0.0-beta < 1.0.0-beta.2 < 1.0.0-beta.11 < 1.0.0-rc.1 < 1.0.0.
//
// Build metadata MAY be denoted by appending a plus sign and a series of dot
// separated identifiers immediately following the patch or pre-release version.
// Build metadata SHOULD be ignored when determining version precedence. Thus
// two versions that differ only in the build metadata, have the same precedence.

package version

import (
	"sync"

	gover "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/fatih/set.v0"
)

const (
	// If needing to edit the iota, please ensure the following:
	// noUpdate = 0
	// hasUpdate = 1
	// hasMUpdate = 2
	noUpdate uint16 = iota
	hasUpdate
	hasMUpdate
	logModule = "version"
)

var (
	// The full version string
	Version = "1.0.10"
	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit string
	Status    *UpdateStatus
)

func init() {
	if GitCommit != "" {
		Version += "+" + GitCommit[:8]
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

	if !s.seedSet.Has(remoteAddr) {
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
		if s.versionStatus == noUpdate {
			s.versionStatus = hasUpdate
		}

		maxVersion, err := gover.NewVersion(s.maxVerSeen)
		if err != nil {
			return err
		}

		if remoteVersion.GreaterThan(maxVersion) {
			s.maxVerSeen = remoteVerStr
		}
	}
	if remoteVersion.Segments()[0] > localVersion.Segments()[0] {
		s.versionStatus = hasMUpdate
	}
	if s.versionStatus != noUpdate {
		log.WithFields(log.Fields{
			"module":          logModule,
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
