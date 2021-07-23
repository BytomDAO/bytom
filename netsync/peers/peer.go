package peers

import (
	"encoding/hex"
	"net"
	"reflect"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/flowrate"
	"gopkg.in/fatih/set.v0"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	msgs "github.com/bytom/bytom/netsync/messages"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

const (
	maxKnownTxs           = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownSignatures    = 1024  // Maximum block signatures to keep in the known list (prevent DOS)
	maxKnownBlocks        = 1024  // Maximum block hashes to keep in the known list (prevent DOS)
	maxFilterAddressSize  = 50
	maxFilterAddressCount = 1000

	logModule = "peers"
)

var (
	errSendStatusMsg = errors.New("send status msg fail")
	ErrPeerMisbehave = errors.New("peer is misbehave")
	ErrNoValidPeer   = errors.New("Can't find valid fast sync peer")
)

//BasePeer is the interface for connection level peer
type BasePeer interface {
	Moniker() string
	Addr() net.Addr
	ID() string
	RemoteAddrHost() string
	ServiceFlag() consensus.ServiceFlag
	TrafficStatus() (*flowrate.Status, *flowrate.Status)
	TrySend(byte, interface{}) bool
	IsLAN() bool
}

//BasePeerSet is the intergace for connection level peer manager
type BasePeerSet interface {
	StopPeerGracefully(string)
	IsBanned(ip string, level byte, reason string) bool
}

type BroadcastMsg interface {
	FilterTargetPeers(ps *PeerSet) []string
	MarkSendRecord(ps *PeerSet, peers []string)
	GetChan() byte
	GetMsg() interface{}
	MsgString() string
}

// PeerInfo indicate peer status snap
type PeerInfo struct {
	ID                  string `json:"peer_id"`
	Moniker             string `json:"moniker"`
	RemoteAddr          string `json:"remote_addr"`
	Height              uint64 `json:"height"`
	Ping                string `json:"ping"`
	Duration            string `json:"duration"`
	TotalSent           int64  `json:"total_sent"`
	TotalReceived       int64  `json:"total_received"`
	AverageSentRate     int64  `json:"average_sent_rate"`
	AverageReceivedRate int64  `json:"average_received_rate"`
	CurrentSentRate     int64  `json:"current_sent_rate"`
	CurrentReceivedRate int64  `json:"current_received_rate"`
}

type Peer struct {
	BasePeer
	mtx             sync.RWMutex
	services        consensus.ServiceFlag
	bestHeight      uint64
	bestHash        *bc.Hash
	justifiedHeight uint64
	justifiedHash   *bc.Hash
	knownTxs        *set.Set // Set of transaction hashes known to be known by this peer
	knownBlocks     *set.Set // Set of block hashes known to be known by this peer
	knownSignatures *set.Set // Set of block signatures known to be known by this peer
	knownStatus     uint64   // Set of chain status known to be known by this peer
	filterAdds      *set.Set // Set of addresses that the spv node cares about.
}

func newPeer(basePeer BasePeer) *Peer {
	return &Peer{
		BasePeer:        basePeer,
		services:        basePeer.ServiceFlag(),
		knownTxs:        set.New(),
		knownBlocks:     set.New(),
		knownSignatures: set.New(),
		filterAdds:      set.New(),
	}
}

func (p *Peer) Height() uint64 {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return p.bestHeight
}

func (p *Peer) JustifiedHeight() uint64 {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return p.justifiedHeight
}

func (p *Peer) AddFilterAddress(address []byte) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if p.filterAdds.Size() >= maxFilterAddressCount {
		log.WithField("module", logModule).Warn("the count of filter addresses is greater than limit")
		return
	}
	if len(address) > maxFilterAddressSize {
		log.WithField("module", logModule).Warn("the size of filter address is greater than limit")
		return
	}

	p.filterAdds.Add(hex.EncodeToString(address))
}

func (p *Peer) AddFilterAddresses(addresses [][]byte) {
	if !p.filterAdds.IsEmpty() {
		p.filterAdds.Clear()
	}
	for _, address := range addresses {
		p.AddFilterAddress(address)
	}
}

func (p *Peer) FilterClear() {
	p.filterAdds.Clear()
}

func (p *Peer) GetBlockByHeight(height uint64) bool {
	msg := struct{ msgs.BlockchainMessage }{&msgs.GetBlockMessage{Height: height}}
	return p.TrySend(msgs.BlockchainChannel, msg)
}

func (p *Peer) GetBlocks(locator []*bc.Hash, stopHash *bc.Hash) bool {
	msg := struct{ msgs.BlockchainMessage }{msgs.NewGetBlocksMessage(locator, stopHash)}
	return p.TrySend(msgs.BlockchainChannel, msg)
}

func (p *Peer) GetHeaders(locator []*bc.Hash, stopHash *bc.Hash, skip uint64) bool {
	msg := struct{ msgs.BlockchainMessage }{msgs.NewGetHeadersMessage(locator, stopHash, skip)}
	return p.TrySend(msgs.BlockchainChannel, msg)
}

func (p *Peer) GetPeerInfo() *PeerInfo {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	sentStatus, receivedStatus := p.TrafficStatus()
	ping := sentStatus.Idle - receivedStatus.Idle
	if receivedStatus.Idle > sentStatus.Idle {
		ping = -ping
	}

	return &PeerInfo{
		ID:                  p.ID(),
		Moniker:             p.BasePeer.Moniker(),
		RemoteAddr:          p.Addr().String(),
		Height:              p.bestHeight,
		Ping:                ping.String(),
		Duration:            sentStatus.Duration.String(),
		TotalSent:           sentStatus.Bytes,
		TotalReceived:       receivedStatus.Bytes,
		AverageSentRate:     sentStatus.AvgRate,
		AverageReceivedRate: receivedStatus.AvgRate,
		CurrentSentRate:     sentStatus.CurRate,
		CurrentReceivedRate: receivedStatus.CurRate,
	}
}

func (p *Peer) getRelatedTxs(txs []*types.Tx) []*types.Tx {
	var relatedTxs []*types.Tx
	for _, tx := range txs {
		if p.isRelatedTx(tx) {
			relatedTxs = append(relatedTxs, tx)
		}
	}
	return relatedTxs
}

func (p *Peer) isRelatedTx(tx *types.Tx) bool {
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

func (p *Peer) isSPVNode() bool {
	return !p.services.IsEnable(consensus.SFFullNode)
}

func (p *Peer) MarkBlock(hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	for p.knownBlocks.Size() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash.String())
}

func (p *Peer) markNewStatus(height uint64) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.knownStatus = height
}

func (p *Peer) markSign(signature []byte) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	for p.knownSignatures.Size() >= maxKnownSignatures {
		p.knownSignatures.Pop()
	}
	p.knownSignatures.Add(hex.EncodeToString(signature))
}

func (p *Peer) markTransaction(hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	for p.knownTxs.Size() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash.String())
}

func (p *Peer) SendBlock(block *types.Block) (bool, error) {
	msg, err := msgs.NewBlockMessage(block)
	if err != nil {
		return false, errors.Wrap(err, "fail on NewBlockMessage")
	}

	ok := p.TrySend(msgs.BlockchainChannel, struct{ msgs.BlockchainMessage }{msg})
	if ok {
		blcokHash := block.Hash()
		p.knownBlocks.Add(blcokHash.String())
	}
	return ok, nil
}

func (p *Peer) SendBlocks(blocks []*types.Block) (bool, error) {
	msg, err := msgs.NewBlocksMessage(blocks)
	if err != nil {
		return false, errors.Wrap(err, "fail on NewBlocksMessage")
	}

	if ok := p.TrySend(msgs.BlockchainChannel, struct{ msgs.BlockchainMessage }{msg}); !ok {
		return ok, nil
	}

	for _, block := range blocks {
		blcokHash := block.Hash()
		p.knownBlocks.Add(blcokHash.String())
	}
	return true, nil
}

func (p *Peer) SendHeaders(headers []*types.BlockHeader) (bool, error) {
	msg, err := msgs.NewHeadersMessage(headers)
	if err != nil {
		return false, errors.New("fail on NewHeadersMessage")
	}

	ok := p.TrySend(msgs.BlockchainChannel, struct{ msgs.BlockchainMessage }{msg})
	return ok, nil
}

func (p *Peer) SendMerkleBlock(block *types.Block) (bool, error) {
	msg := msgs.NewMerkleBlockMessage()
	if err := msg.SetRawBlockHeader(block.BlockHeader); err != nil {
		return false, err
	}

	relatedTxs := p.getRelatedTxs(block.Transactions)

	txHashes, txFlags := types.GetTxMerkleTreeProof(block.Transactions, relatedTxs)
	if err := msg.SetTxInfo(txHashes, txFlags, relatedTxs); err != nil {
		return false, nil
	}

	ok := p.TrySend(msgs.BlockchainChannel, struct{ msgs.BlockchainMessage }{msg})
	return ok, nil
}

func (p *Peer) SendTransactions(txs []*types.Tx) error {
	validTxs := make([]*types.Tx, 0, len(txs))
	for i, tx := range txs {
		if p.isSPVNode() && !p.isRelatedTx(tx) || p.knownTxs.Has(tx.ID.String()) {
			continue
		}

		validTxs = append(validTxs, tx)
		if len(validTxs) != msgs.TxsMsgMaxTxNum && i != len(txs)-1 {
			continue
		}

		msg, err := msgs.NewTransactionsMessage(validTxs)
		if err != nil {
			return err
		}

		if ok := p.TrySend(msgs.BlockchainChannel, struct{ msgs.BlockchainMessage }{msg}); !ok {
			return errors.New("failed to send txs msg")
		}

		for _, validTx := range validTxs {
			p.knownTxs.Add(validTx.ID.String())
		}

		validTxs = make([]*types.Tx, 0, len(txs))
	}

	return nil
}

func (p *Peer) SendStatus(bestHeader, justifiedHeader *types.BlockHeader) error {
	msg := msgs.NewStatusMessage(bestHeader, justifiedHeader)
	if ok := p.TrySend(msgs.BlockchainChannel, struct{ msgs.BlockchainMessage }{msg}); !ok {
		return errSendStatusMsg
	}
	p.markNewStatus(bestHeader.Height)
	return nil
}

func (p *Peer) SetBestStatus(bestHeight uint64, bestHash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.bestHeight = bestHeight
	p.bestHash = bestHash
}

func (p *Peer) SetJustifiedStatus(justifiedHeight uint64, justifiedHash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.justifiedHeight = justifiedHeight
	p.justifiedHash = justifiedHash
}

type PeerSet struct {
	BasePeerSet
	mtx   sync.RWMutex
	peers map[string]*Peer
}

// newPeerSet creates a new peer set to track the active participants.
func NewPeerSet(basePeerSet BasePeerSet) *PeerSet {
	return &PeerSet{
		BasePeerSet: basePeerSet,
		peers:       make(map[string]*Peer),
	}
}

func (ps *PeerSet) ProcessIllegal(peerID string, level byte, reason string) {
	ps.mtx.Lock()
	peer := ps.peers[peerID]
	ps.mtx.Unlock()

	if peer == nil {
		return
	}

	if banned := ps.IsBanned(peer.RemoteAddrHost(), level, reason); banned {
		ps.RemovePeer(peerID)
	}
	return
}

func (ps *PeerSet) AddPeer(peer BasePeer) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if _, ok := ps.peers[peer.ID()]; !ok {
		ps.peers[peer.ID()] = newPeer(peer)
		return
	}
	log.WithField("module", logModule).Warning("add existing peer to blockKeeper")
}

func (ps *PeerSet) BestPeer(flag consensus.ServiceFlag) *Peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	var bestPeer *Peer
	for _, p := range ps.peers {
		if !p.services.IsEnable(flag) {
			continue
		}
		if bestPeer == nil || p.JustifiedHeight() > bestPeer.JustifiedHeight() ||
			(p.JustifiedHeight() == bestPeer.JustifiedHeight() && p.bestHeight > bestPeer.bestHeight) ||
			(p.JustifiedHeight() == bestPeer.JustifiedHeight() && p.bestHeight == bestPeer.bestHeight && p.IsLAN()) {
			bestPeer = p
		}
	}
	return bestPeer
}

//SendMsg send message to the target peer.
func (ps *PeerSet) SendMsg(peerID string, msgChannel byte, msg interface{}) bool {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return false
	}

	ok := peer.TrySend(msgChannel, msg)
	if !ok {
		ps.RemovePeer(peerID)
	}
	return ok
}

//BroadcastMsg Broadcast message to the target peers
// and mark the message send record
func (ps *PeerSet) BroadcastMsg(bm BroadcastMsg) error {
	//filter target peers
	peers := bm.FilterTargetPeers(ps)

	//broadcast to target peers
	peersSuccess := make([]string, 0)
	for _, peer := range peers {
		if ok := ps.SendMsg(peer, bm.GetChan(), bm.GetMsg()); !ok {
			log.WithFields(log.Fields{"module": logModule, "peer": peer, "type": reflect.TypeOf(bm.GetMsg()), "message": bm.MsgString()}).Warning("send message to peer error")
			continue
		}
		peersSuccess = append(peersSuccess, peer)
	}

	//mark the message send record
	bm.MarkSendRecord(ps, peersSuccess)
	return nil
}

func (ps *PeerSet) BroadcastNewStatus(bestHeader, justifiedHeader *types.BlockHeader) error {
	msg := msgs.NewStatusMessage(bestHeader, justifiedHeader)
	peers := ps.peersWithoutNewStatus(bestHeader.Height)
	for _, peer := range peers {
		if ok := peer.TrySend(msgs.BlockchainChannel, struct{ msgs.BlockchainMessage }{msg}); !ok {
			ps.RemovePeer(peer.ID())
			continue
		}

		peer.markNewStatus(bestHeader.Height)
	}
	return nil
}

func (ps *PeerSet) BroadcastTx(tx *types.Tx) error {
	msg, err := msgs.NewTransactionMessage(tx)
	if err != nil {
		return errors.Wrap(err, "fail on broadcast tx")
	}

	peers := ps.peersWithoutTx(&tx.ID)
	for _, peer := range peers {
		if peer.isSPVNode() && !peer.isRelatedTx(tx) {
			continue
		}
		if ok := peer.TrySend(msgs.BlockchainChannel, struct{ msgs.BlockchainMessage }{msg}); !ok {
			log.WithFields(log.Fields{
				"module":  logModule,
				"peer":    peer.Addr(),
				"type":    reflect.TypeOf(msg),
				"message": msg.String(),
			}).Warning("send message to peer error")
			ps.RemovePeer(peer.ID())
			continue
		}
		peer.markTransaction(&tx.ID)
	}
	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *PeerSet) GetPeer(id string) *Peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return ps.peers[id]
}

func (ps *PeerSet) GetPeersByHeight(height uint64) []*Peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	peers := []*Peer{}
	for _, peer := range ps.peers {
		if peer.Height() >= height {
			peers = append(peers, peer)
		}
	}
	return peers
}

func (ps *PeerSet) GetPeerInfos() []*PeerInfo {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	result := []*PeerInfo{}
	for _, peer := range ps.peers {
		result = append(result, peer.GetPeerInfo())
	}
	return result
}

func (ps *PeerSet) MarkBlock(peerID string, hash *bc.Hash) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}
	peer.MarkBlock(hash)
}

func (ps *PeerSet) MarkBlockVerification(peerID string, signature []byte) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}
	peer.markSign(signature)
}

func (ps *PeerSet) MarkStatus(peerID string, height uint64) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}
	peer.markNewStatus(height)
}

func (ps *PeerSet) MarkTx(peerID string, txHash bc.Hash) {
	ps.mtx.Lock()
	peer := ps.peers[peerID]
	ps.mtx.Unlock()

	if peer == nil {
		return
	}
	peer.markTransaction(&txHash)
}

func (ps *PeerSet) PeersWithoutBlock(hash bc.Hash) []string {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	var peers []string
	for _, peer := range ps.peers {
		if !peer.knownBlocks.Has(hash.String()) {
			peers = append(peers, peer.ID())
		}
	}
	return peers
}

func (ps *PeerSet) PeersWithoutSignature(signature []byte) []string {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	var peers []string
	for _, peer := range ps.peers {
		if !peer.knownSignatures.Has(hex.EncodeToString(signature)) {
			peers = append(peers, peer.ID())
		}
	}
	return peers
}

func (ps *PeerSet) peersWithoutNewStatus(height uint64) []*Peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	var peers []*Peer
	for _, peer := range ps.peers {
		if peer.knownStatus < height {
			peers = append(peers, peer)
		}
	}
	return peers
}

func (ps *PeerSet) peersWithoutTx(hash *bc.Hash) []*Peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	peers := []*Peer{}
	for _, peer := range ps.peers {
		if !peer.knownTxs.Has(hash.String()) {
			peers = append(peers, peer)
		}
	}
	return peers
}

func (ps *PeerSet) RemovePeer(peerID string) {
	ps.mtx.Lock()
	delete(ps.peers, peerID)
	ps.mtx.Unlock()
	ps.StopPeerGracefully(peerID)
}

func (ps *PeerSet) SetStatus(peerID string, height uint64, hash *bc.Hash) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}

	peer.SetBestStatus(height, hash)
}

func (ps *PeerSet) SetJustifiedStatus(peerID string, height uint64, hash *bc.Hash) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}

	peer.SetJustifiedStatus(height, hash)
}
