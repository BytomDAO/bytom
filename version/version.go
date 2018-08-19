package version

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
	maxVerSeen    *VerNum
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
	maxVerSeen, _ = parse(Version)
}

/* 						Functions for version-control					*/
// -------------------------------start----------------------------------

// CompatibleWith checks whether the remote peer version is compatible with the
// node itself.
// RULES:
// | local |       remote        |
// |   -   |         -           |
// | 1.0.4 | same major version. |
func CompatibleWith(remoteVer string) (bool, error) {
	localVerNum, err := parse(Version)
	if err != nil {
		return false, err
	}
	remoteVerNum, err := parse(remoteVer)
	if err != nil {
		return false, err
	}
	return (localVerNum.major == remoteVerNum.major), nil
}

// Deprecate checks whether a remote peer version is too old and should be
// deprecated.
// It aims at providing support for CheckUpdateRequestMessage & CheckUpdateResponseMessage,
// and should only serve on bytomd seed nodes.
// RULES:
// | local |       remote        |
// |   -   |         -           |
// | 1.0.4 |      below 1.0.0    |
func Deprecate(remoteVer string) (bool, error) {
	limitVerNum, err := parse(deprecateBelow)
	if err != nil {
		return false, err
	}
	remoteVerNum, err := parse(remoteVer)
	if err != nil {
		return false, err
	}

	return limitVerNum.greaterThan(remoteVerNum)
}

// OlderThan checks whether the node version is older than a remote peer.
// remoteVer is supposed to always be corresponding to a seed
func OlderThan(remoteVer string) (bool, error) {
	localVerNum, err := parse(Version)
	if err != nil {
		return false, err
	}
	remoteVerNum, err := parse(remoteVer)
	if err != nil {
		return false, err
	}

	// Reset notifiedTimes
	if greaterThanMax, err := remoteVerNum.greaterThan(maxVerSeen); (err == nil) && greaterThanMax {
		maxVerSeen = remoteVerNum
		notifiedTimes = uint16(0)
	}

	return remoteVerNum.greaterThan(localVerNum)
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
