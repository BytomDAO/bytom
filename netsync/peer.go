package netsync

import (
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/fatih/set.v0"

	"github.com/bytom/consensus"
	"github.com/bytom/errors"
	"github.com/bytom/p2p"
	"github.com/bytom/p2p/trust"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	maxKnownTxs         = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks      = 1024  // Maximum block hashes to keep in the known list (prevent DOS)
	defaultBanThreshold = uint64(100)
)

//BasePeer is the interface for connection level peer
type BasePeer interface {
	Addr() net.Addr
	CloseConn()
	ID() string
	ServiceFlag() consensus.ServiceFlag
	TrySend(byte, interface{}) bool
}

//BasePeerSet is the intergace for connection level peer manager
type BasePeerSet interface {
	AddBannedPeer(string) error
	StopPeerGracefully(string)
}

type peer struct {
	BasePeer
	mtx         sync.RWMutex
	services    consensus.ServiceFlag
	height      uint64
	hash        *bc.Hash
	banScore    trust.DynamicBanScore
	knownTxs    *set.Set // Set of transaction hashes known to be known by this peer
	knownBlocks *set.Set // Set of block hashes known to be known by this peer
}

func newPeer(height uint64, hash *bc.Hash, basePeer BasePeer) *peer {
	return &peer{
		BasePeer:    basePeer,
		services:    basePeer.ServiceFlag(),
		height:      height,
		hash:        hash,
		knownTxs:    set.New(),
		knownBlocks: set.New(),
	}
}

func (p *peer) Height() uint64 {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.height
}

func (p *peer) addBanScore(persistent, transient uint64, reason string) bool {
	score := p.banScore.Increase(persistent, transient)
	if score > defaultBanThreshold {
		log.WithFields(log.Fields{"address": p.Addr(), "score": score, "reason": reason}).Errorf("banning and disconnecting")
		return true
	}

	warnThreshold := defaultBanThreshold >> 1
	if score > warnThreshold {
		log.WithFields(log.Fields{"address": p.Addr(), "score": score, "reason": reason}).Warning("ban score increasing")
	}
	return false
}

func (p *peer) getBlockByHeight(height uint64) bool {
	msg := struct{ BlockchainMessage }{&GetBlockMessage{Height: height}}
	return p.TrySend(BlockchainChannel, msg)
}

func (p *peer) getBlocks(locator []*bc.Hash, stopHash *bc.Hash) bool {
	msg := struct{ BlockchainMessage }{NewGetBlocksMessage(locator, stopHash)}
	return p.TrySend(BlockchainChannel, msg)
}

func (p *peer) getHeaders(locator []*bc.Hash, stopHash *bc.Hash) bool {
	msg := struct{ BlockchainMessage }{NewGetHeadersMessage(locator, stopHash)}
	return p.TrySend(BlockchainChannel, msg)
}

func (p *peer) markBlock(hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	for p.knownBlocks.Size() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash.String())
}

func (p *peer) markTransaction(hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	for p.knownTxs.Size() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash.String())
}

func (p *peer) sendBlock(block *types.Block) error {
	msg, err := NewBlockMessage(block)
	if err != nil {
		return errors.Wrap(err, "fail on NewBlockMessage")
	}

	p.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}

func (p *peer) sendBlocks(blocks []*types.Block) error {
	msg, err := NewBlocksMessage(blocks)
	if err != nil {
		return errors.Wrap(err, "fail on NewBlocksMessage")
	}

	p.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}

func (p *peer) sendHeaders(headers []*types.BlockHeader) error {
	msg, err := NewHeadersMessage(headers)
	if err != nil {
		return errors.New("fail on NewHeadersMessage")
	}

	p.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}

func (p *peer) sendStatus(blockHeader *types.BlockHeader, genesis *bc.Hash) bool {
	msg := NewStatusResponseMessage(blockHeader, genesis)
	return p.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
}

func (p *peer) sendTransactions(txs []*types.Tx) error {
	for _, tx := range txs {
		msg, err := NewTransactionMessage(tx)
		if err != nil {
			return errors.Wrap(err, "failed to tx msg")
		}

		p.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
		p.knownTxs.Add(tx.ID.String())
	}
	return nil
}

func (p *peer) setStatus(height uint64, hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.height = height
	p.hash = hash
}

type peerSet struct {
	BasePeerSet
	mtx   sync.RWMutex
	peers map[string]*peer
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet(basePeerSet BasePeerSet) *peerSet {
	return &peerSet{
		BasePeerSet: basePeerSet,
		peers:       make(map[string]*peer),
	}
}

func (ps *peerSet) BestPeer(flag consensus.ServiceFlag) *peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	var bestPeer *peer
	for _, p := range ps.peers {
		if !p.services.IsEnable(flag) {
			continue
		}
		if bestPeer == nil || p.height > bestPeer.height {
			bestPeer = p
		}
	}
	return bestPeer
}

func (ps *peerSet) addPeer(peer *p2p.Peer, height uint64, hash *bc.Hash) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if _, ok := ps.peers[peer.Key]; !ok {
		ps.peers[peer.Key] = newPeer(height, hash, peer)
		return
	}
	log.WithField("ID", peer.Key).Warning("add existing peer to blockKeeper")
}

func (ps *peerSet) broadcastMinedBlock(block *types.Block) error {
	msg, err := NewMinedBlockMessage(block)
	if err != nil {
		return errors.Wrap(err, "fail on broadcast mined block")
	}

	hash := block.Hash()
	peers := ps.peersWithoutBlock(&hash)
	for _, peer := range peers {
		if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
			ps.StopPeerGracefully(peer.ID())
			continue
		}
		peer.markBlock(&hash)
	}
	return nil
}

func (ps *peerSet) broadcastTx(tx *types.Tx) error {
	msg, err := NewTransactionMessage(tx)
	if err != nil {
		return errors.Wrap(err, "fail on broadcast tx")
	}

	peers := ps.peersWithoutTx(&tx.ID)
	for _, peer := range peers {
		if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
			ps.StopPeerGracefully(peer.ID())
			continue
		}
		peer.markTransaction(&tx.ID)
	}
	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) getPeer(id string) *peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) len() int {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return len(ps.peers)
}

func (ps *peerSet) peersWithoutBlock(hash *bc.Hash) []*peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	peers := []*peer{}
	for _, peer := range ps.peers {
		if !peer.knownBlocks.Has(hash.String()) {
			peers = append(peers, peer)
		}
	}
	return peers
}

func (ps *peerSet) peersWithoutTx(hash *bc.Hash) []*peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	peers := []*peer{}
	for _, peer := range ps.peers {
		if !peer.knownTxs.Has(hash.String()) {
			peers = append(peers, peer)
		}
	}
	return peers
}

func (ps *peerSet) removePeer(peerID string) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	delete(ps.peers, peerID)
}
