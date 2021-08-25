package common

const (
	NodeUnknownStatus uint8 = iota
	NodeHealthyStatus
	NodeCongestedStatus
	NodeOrphanStatus
	NodeOfflineStatus
)

var StatusLookupTable = map[uint8]string{
	NodeUnknownStatus:   "unknown",
	NodeHealthyStatus:   "healthy",
	NodeCongestedStatus: "congested",
	NodeOrphanStatus:    "orphan",
	NodeOfflineStatus:   "offline",
}
