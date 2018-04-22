package node

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/fatih/set.v0"

	"github.com/bytom/p2p"
	"github.com/bytom/p2p/trust"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	defaultVersion      = 1
	defaultBanThreshold = uint64(100)

	maxKnownTxs    = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks = 1024  // Maximum block hashes to keep in the known list (prevent DOS)
)

type peer struct {
	mtx      sync.RWMutex
	version  int // Protocol version negotiated
	id       string
	height   uint64
	hash     *bc.Hash
	banScore trust.DynamicBanScore

	swPeer *p2p.Peer

	knownTxs    *set.Set // Set of transaction hashes known to be known by this peer
	knownBlocks *set.Set // Set of block hashes known to be known by this peer
}

func newPeer(height uint64, hash *bc.Hash, Peer *p2p.Peer) *peer {
	return &peer{
		version:     defaultVersion,
		id:          Peer.Key,
		height:      height,
		hash:        hash,
		swPeer:      Peer,
		knownTxs:    set.New(),
		knownBlocks: set.New(),
	}
}

func (p *peer) setStatus(height uint64, hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.height = height
	p.hash = hash
}

func (p *peer) reqBlockByHeight(height uint64) bool {
	msg := &BlockRequestMessage{Height: height}
	return p.swPeer.Send(BlockchainChannel, struct{ BlockchainMessage }{msg})
}

func (p *peer) sendTransaction(tx *types.Tx) bool {
	if p.knownTxs.Has(tx.ID.String()) {
		return true
	}

	msg, err := NewTransactionNotifyMessage(tx)
	if err != nil {
		log.WithField("err", err).Errorf("NewTransactionNotifyMessage fail on generate message")
		return true
	}

	p.markTransaction(&tx.ID)
	return p.swPeer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
}

func (p *peer) sendBlock(block *types.Block) bool {
	blockHash := block.Hash()
	if p.knownBlocks.Has(blockHash.String()) {
		return true
	}

	msg, err := NewMinedBlockMessage(block)
	if err != nil {
		log.WithField("err", err).Errorf("NewMinedBlockMessage fail on generate message")
		return true
	}

	p.markBlock(&blockHash)
	return p.swPeer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
}

func (p *peer) getPeer() *p2p.Peer {
	return p.swPeer
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) markTransaction(hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownTxs.Size() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash.String())
}

// MarkBlock marks a block as known for the peer, ensuring that the block will
// never be propagated to this particular peer.
func (p *peer) markBlock(hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	// If we reached the memory allowance, drop a previously known block hash
	for p.knownBlocks.Size() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash.String())
}

// addBanScore increases the persistent and decaying ban score fields by the
// values passed as parameters. If the resulting score exceeds half of the ban
// threshold, a warning is logged including the reason provided. Further, if
// the score is above the ban threshold, the peer will be banned and
// disconnected.
func (p *peer) addBanScore(persistent, transient uint64, reason string) bool {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	warnThreshold := defaultBanThreshold >> 1
	if transient == 0 && persistent == 0 {
		// The score is not being increased, but a warning message is still
		// logged if the score is above the warn threshold.
		score := p.banScore.Int()
		if score > warnThreshold {
			log.Infof("Misbehaving peer %s: %s -- ban score is %d, "+"it was not increased this time", p.id, reason, score)
		}
		return false
	}
	score := p.banScore.Increase(persistent, transient)
	if score > warnThreshold {
		log.Infof("Misbehaving peer %s: %s -- ban score increased to %d", p.id, reason, score)
		if score > defaultBanThreshold {
			log.Errorf("Misbehaving peer %s -- banning and disconnecting", p.id)
			return true
		}
	}
	return false
}
