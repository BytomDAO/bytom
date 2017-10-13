package core

import (
	"github.com/bytom/blockchain/txdb"
	p2p "github.com/bytom/p2p"
	"github.com/bytom/types"
	"github.com/tendermint/tmlibs/log"
)

type P2P interface {
	Listeners() []p2p.Listener
	Peers() p2p.IPeerSet
	NumPeers() (outbound, inbound, dialig int)
	NodeInfo() *p2p.NodeInfo
	IsListening() bool
	DialSeeds(*p2p.AddrBook, []string) error
}

var (
	// external, thread safe interfaces
	eventSwitch types.EventSwitch
	blockStore  *txdb.Store
	p2pSwitch   P2P

	addrBook *p2p.AddrBook

	logger log.Logger
)

func SetEventSwitch(evsw types.EventSwitch) {
	eventSwitch = evsw
}

func SetBlockStore(bs *txdb.Store) {
	blockStore = bs
}

func SetSwitch(sw P2P) {
	p2pSwitch = sw
}

func SetAddrBook(book *p2p.AddrBook) {
	addrBook = book
}

func SetLogger(l log.Logger) {
	logger = l
}
