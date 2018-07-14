package netsync

import (
	"net"

	"github.com/bytom/consensus"
)

type BasePeer interface {
	Addr() net.Addr
	CloseConn()
	ID() string
	ServiceFlag() consensus.ServiceFlag
	TrySend(byte, interface{}) bool
}

type BasePeerSet interface {
	AddBannedPeer(string) error
	StopPeerGracefully(string)
}
