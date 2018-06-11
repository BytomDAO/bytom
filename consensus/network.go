package consensus

// ServiceFlag use uint64 to indicate what kind of server this node can provide.
// one uint64 can represent 64 type of service flag
type ServiceFlag uint64

const (
	// SFFullNode is a flag used to indicate a peer is a full node.
	SFFullNode ServiceFlag = 1 << iota

	DefaultServices = SFFullNode
)
