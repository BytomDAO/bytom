package blockchain

import (
	"errors"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/p2p"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

type blockKeeperPeer struct {
	mtx    sync.RWMutex
	height uint64
	hash   *bc.Hash
}

func newBlockKeeperPeer(height uint64, hash *bc.Hash) *blockKeeperPeer {
	return &blockKeeperPeer{
		height: height,
		hash:   hash,
	}
}

func (p *blockKeeperPeer) GetStatus() (height uint64, hash *bc.Hash) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.height, p.hash
}

func (p *blockKeeperPeer) SetStatus(height uint64, hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.height = height
	p.hash = hash
}

type pendingResponse struct {
	block  *legacy.Block
	peerID string
}

type blockKeeper struct {
	mtx              sync.RWMutex
	chain            *protocol.Chain
	sw               *p2p.Switch
	peers            map[string]*blockKeeperPeer
	pendingProcessCh chan *pendingResponse
}

func newBlockKeeper(chain *protocol.Chain, sw *p2p.Switch) *blockKeeper {
	return &blockKeeper{
		chain:            chain,
		sw:               sw,
		peers:            make(map[string]*blockKeeperPeer),
		pendingProcessCh: make(chan *pendingResponse),
	}
}

func (bk *blockKeeper) AddBlock(block *legacy.Block, peerID string) {
	bk.pendingProcessCh <- &pendingResponse{block: block, peerID: peerID}
}

func (bk *blockKeeper) IsCaughtUp() bool {
	bk.mtx.RLock()
	defer bk.mtx.RUnlock()
	if len(bk.peers) == 0 {
		log.Debug("IsCaughtUp: no peer in the blockKeeper")
		return true
	}

	selfHeight := bk.chain.Height()
	maxPeerHeight := uint64(0)
	for _, peer := range bk.peers {
		peerHeight, _ := peer.GetStatus()
		if peerHeight > maxPeerHeight {
			maxPeerHeight = peerHeight
		}
	}

	isCaughtUp := selfHeight >= maxPeerHeight
	log.WithFields(log.Fields{"height": selfHeight, "maxPeerHeight": maxPeerHeight}).Infof("IsCaughtUp: %v", isCaughtUp)
	return isCaughtUp
}

func (bk *blockKeeper) RemovePeer(peerID string) {
	bk.mtx.Lock()
	defer bk.mtx.Unlock()
	delete(bk.peers, peerID)
}

func (bk *blockKeeper) RequestBlockByHash(peerID string, hash *bc.Hash) error {
	peer := bk.sw.Peers().Get(peerID)
	if peer == nil {
		return errors.New("can't find peer in peer pool")
	}
	msg := &BlockRequestMessage{RawHash: hash.Byte32()}
	peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}

func (bk *blockKeeper) RequestBlockByHeight(peerID string, height uint64) error {
	peer := bk.sw.Peers().Get(peerID)
	if peer == nil {
		return errors.New("can't find peer in peer pool")
	}
	msg := &BlockRequestMessage{Height: height}
	peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}

func (bk *blockKeeper) SetPeerHeight(peerID string, height uint64, hash *bc.Hash) {
	if peer, ok := bk.peers[peerID]; ok {
		peer.SetStatus(height, hash)
		return
	}

	peer := newBlockKeeperPeer(height, hash)
	bk.mtx.Lock()
	bk.peers[peerID] = peer
	bk.mtx.Unlock()
}

func (bk *blockKeeper) blockUpdater() {
	bk.mtx.RLock()
	defer bk.mtx.RUnlock()
	if len(bk.peers) == 0 {
		return
	}

	selfHeight := bk.chain.Height()
	for peerID, peer := range bk.peers {
		if peerHeight, _ := peer.GetStatus(); peerHeight < bk.chain.Height() {
			continue
		}
		if err := bk.RequestBlockByHeight(peerID, selfHeight+1); err == nil {
			return
		}
	}
}

func (bk *blockKeeper) blockProcesser() {
	for pendingResponse := range bk.pendingProcessCh {
		block := pendingResponse.block
		isOrphan, err := bk.chain.ProcessBlock(block)
		if err != nil {
			blockHash := block.Hash()
			log.WithField("hash", blockHash.String()).Errorf("blockKeeper fail process block %v", err)
		}
		if !isOrphan {
			continue
		}
		bk.RequestBlockByHash(pendingResponse.peerID, &block.PreviousBlockHash)
	}
}
