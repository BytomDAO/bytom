package netsync

import (
	"encoding/hex"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/fatih/set.v0"

	"github.com/bytom/consensus"
	"github.com/bytom/errors"
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
	ID() string
	ServiceFlag() consensus.ServiceFlag
	TrySend(byte, interface{}) bool
}

//BasePeerSet is the intergace for connection level peer manager
type BasePeerSet interface {
	AddBannedPeer(string) error
	StopPeerGracefully(string)
}

// PeerInfo indicate peer status snap
type PeerInfo struct {
	ID         string `json:"peer_id"`
	RemoteAddr string `json:"remote_addr"`
	Height     uint64 `json:"height"`
	Delay      uint32 `json:"delay"`
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
	filterAdds  *set.Set // Set of addresses that the spv node cares about.
}

func newPeer(height uint64, hash *bc.Hash, basePeer BasePeer) *peer {
	return &peer{
		BasePeer:    basePeer,
		services:    basePeer.ServiceFlag(),
		height:      height,
		hash:        hash,
		knownTxs:    set.New(),
		knownBlocks: set.New(),
		filterAdds:  set.New(),
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

func (p *peer) addFilterAddress(address []byte) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if p.filterAdds.Size() >= maxFilterAddressCount {
		log.Warn("the count of filter addresses is greater than limit")
		return
	}
	if len(address) > maxFilterAddressSize {
		log.Warn("the size of filter address is greater than limit")
		return
	}
	p.filterAdds.Add(hex.EncodeToString(address))
}

func (p *peer) addFilterAddresses(addresses [][]byte) {
	if !p.filterAdds.IsEmpty() {
		p.filterAdds.Clear()
	}
	for _, address := range addresses {
		p.addFilterAddress(address)
	}
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

func (p *peer) getPeerInfo() *PeerInfo {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return &PeerInfo{
		ID:         p.ID(),
		RemoteAddr: p.Addr().String(),
		Height:     p.height,
	}
}

func (p *peer) getRelatedTxAndStatus(txs []*types.Tx, txStatuses *bc.TransactionStatus) ([]*types.Tx, []*bc.TxVerifyResult) {
	var relatedTxs []*types.Tx
	var relatedStatuses []*bc.TxVerifyResult
	for i, tx := range txs {
		if p.isRelatedTx(tx) {
			relatedTxs = append(relatedTxs, tx)
			relatedStatuses = append(relatedStatuses, txStatuses.VerifyStatus[i])
		}
	}
	return relatedTxs, relatedStatuses
}

func (p *peer) isRelatedTx(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		switch inp := input.TypedInput.(type) {
		case *types.SpendInput:
			if p.filterAdds.Has(hex.EncodeToString(inp.ControlProgram)) {
				return true
			}
		}
	}
	for _, output := range tx.Outputs {
		if p.filterAdds.Has(hex.EncodeToString(output.ControlProgram)) {
			return true
		}
	}
	return false
}

func (p *peer) isSPVNode() bool {
	return !p.services.IsEnable(consensus.SFFullNode)
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

func (p *peer) sendBlock(block *types.Block) (bool, error) {
	msg, err := NewBlockMessage(block)
	if err != nil {
		return false, errors.Wrap(err, "fail on NewBlockMessage")
	}

	ok := p.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	if ok {
		blcokHash := block.Hash()
		p.knownBlocks.Add(blcokHash.String())
	}
	return ok, nil
}

func (p *peer) sendBlocks(blocks []*types.Block) (bool, error) {
	msg, err := NewBlocksMessage(blocks)
	if err != nil {
		return false, errors.Wrap(err, "fail on NewBlocksMessage")
	}

	if ok := p.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
		return ok, nil
	}

	for _, block := range blocks {
		blcokHash := block.Hash()
		p.knownBlocks.Add(blcokHash.String())
	}
	return true, nil
}

func (p *peer) sendHeaders(headers []*types.BlockHeader) (bool, error) {
	msg, err := NewHeadersMessage(headers)
	if err != nil {
		return false, errors.New("fail on NewHeadersMessage")
	}

	ok := p.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return ok, nil
}

func (p *peer) sendMerkleBlock(block *types.Block, txStatuses *bc.TransactionStatus) (bool, error) {
	msg := NewMerkleBlockMessage()
	if err := msg.setRawBlockHeader(block.BlockHeader); err != nil {
		return false, err
	}

	relatedTxs, relatedStatuses := p.getRelatedTxAndStatus(block.Transactions, txStatuses)

	txHashes, txFlags := types.GetTxMerkleTreeProof(block.Transactions, relatedTxs)
	if err := msg.setTxInfo(txHashes, txFlags, relatedTxs); err != nil {
		return false, nil
	}

	statusHashes := types.GetStatusMerkleTreeProof(txStatuses.VerifyStatus, txFlags)
	if err := msg.setStatusInfo(statusHashes, relatedStatuses); err != nil {
		return false, nil
	}

	ok := p.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return ok, nil
}

func (p *peer) sendTransactions(txs []*types.Tx) (bool, error) {
	for _, tx := range txs {
		if p.isSPVNode() && !p.isRelatedTx(tx) {
			continue
		}
		msg, err := NewTransactionMessage(tx)
		if err != nil {
			return false, errors.Wrap(err, "failed to tx msg")
		}

		if p.knownTxs.Has(tx.ID.String()) {
			continue
		}
		if ok := p.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
			return ok, nil
		}
		p.knownTxs.Add(tx.ID.String())
	}
	return true, nil
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

func (ps *peerSet) addBanScore(peerID string, persistent, transient uint64, reason string) {
	ps.mtx.Lock()
	peer := ps.peers[peerID]
	ps.mtx.Unlock()

	if peer == nil {
		return
	}
	if ban := peer.addBanScore(persistent, transient, reason); !ban {
		return
	}
	if err := ps.AddBannedPeer(peer.Addr().String()); err != nil {
		log.WithField("err", err).Error("fail on add ban peer")
	}
	ps.removePeer(peerID)
}

func (ps *peerSet) addPeer(peer BasePeer, height uint64, hash *bc.Hash) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if _, ok := ps.peers[peer.ID()]; !ok {
		ps.peers[peer.ID()] = newPeer(height, hash, peer)
		return
	}
	log.WithField("ID", peer.ID()).Warning("add existing peer to blockKeeper")
}

func (ps *peerSet) bestPeer(flag consensus.ServiceFlag) *peer {
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

func (ps *peerSet) broadcastMinedBlock(block *types.Block) error {
	msg, err := NewMinedBlockMessage(block)
	if err != nil {
		return errors.Wrap(err, "fail on broadcast mined block")
	}

	hash := block.Hash()
	peers := ps.peersWithoutBlock(&hash)
	for _, peer := range peers {
		if peer.isSPVNode() {
			continue
		}
		if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
			ps.removePeer(peer.ID())
			continue
		}
		peer.markBlock(&hash)
	}
	return nil
}

func (ps *peerSet) broadcastNewStatus(bestBlock, genesisBlock *types.Block) error {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	genesisHash := genesisBlock.Hash()
	msg := NewStatusResponseMessage(&bestBlock.BlockHeader, &genesisHash)
	for _, peer := range ps.peers {
		if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
			ps.removePeer(peer.ID())
			continue
		}
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
		if peer.isSPVNode() && !peer.isRelatedTx(tx) {
			continue
		}
		if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
			ps.removePeer(peer.ID())
			continue
		}
		peer.markTransaction(&tx.ID)
	}
	return nil
}

func (ps *peerSet) errorHandler(peerID string, err error) {
	if errors.Root(err) == errPeerMisbehave {
		ps.addBanScore(peerID, 20, 0, err.Error())
	} else {
		ps.removePeer(peerID)
	}
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) getPeer(id string) *peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return ps.peers[id]
}

func (ps *peerSet) getPeerInfos() []*PeerInfo {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	result := []*PeerInfo{}
	for _, peer := range ps.peers {
		result = append(result, peer.getPeerInfo())
	}
	return result
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
	delete(ps.peers, peerID)
	ps.mtx.Unlock()
	ps.StopPeerGracefully(peerID)
}
