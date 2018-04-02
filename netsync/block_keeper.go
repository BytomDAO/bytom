package netsync

import (
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
	"github.com/bytom/p2p"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	maxKnownTxs    = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks = 1024  // Maximum block hashes to keep in the known list (prevent DOS)

	syncTimeout        = 30 * time.Second
	requestRetryTicker = 15 * time.Second

	maxBlocksPending = 1024
	maxtxsPending    = 32768
	maxQuitReq       = 256
)

var (
	errGetBlockTimeout = errors.New("Get block Timeout")
	errPeerDropped     = errors.New("Peer dropped")
	errCommAbnorm      = errors.New("Peer communication abnormality")
	errScamPeer        = errors.New("Scam peer")
)

type BlockRequestMessage struct {
	Height  uint64
	RawHash [32]byte
}

type pendingResponse struct {
	block  *types.Block
	peerID string
}

type txsNotify struct {
	tx     *types.Tx
	peerID string
}

//TODO: add retry mechanism
type blockKeeper struct {
	chain *protocol.Chain
	sw    *p2p.Switch
	peers *peerSet

	pendingProcessCh chan *pendingResponse
	txsProcessCh     chan *txsNotify
	quitReqBlockCh   chan *string

	mtx sync.RWMutex
}

func newBlockKeeper(chain *protocol.Chain, sw *p2p.Switch, peers *peerSet) *blockKeeper {
	bk := &blockKeeper{
		chain:            chain,
		sw:               sw,
		peers:            peers,
		pendingProcessCh: make(chan *pendingResponse, maxBlocksPending),
		txsProcessCh:     make(chan *txsNotify, maxtxsPending),
		quitReqBlockCh:   make(chan *string, maxQuitReq),
	}
	go bk.txsProcessWorker()
	return bk
}

func (bk *blockKeeper) AddBlock(block *types.Block, peerID string) {
	bk.pendingProcessCh <- &pendingResponse{block: block, peerID: peerID}
}

func (bk *blockKeeper) AddTX(tx *types.Tx, peerID string) {
	bk.txsProcessCh <- &txsNotify{tx: tx, peerID: peerID}
}

func (bk *blockKeeper) IsCaughtUp() bool {
	bk.mtx.RLock()
	defer bk.mtx.RUnlock()
	_, height := bk.peers.BestPeer()
	return bk.chain.Height() < height
}

func (bk *blockKeeper) RemovePeer(peerID string) {
	bk.mtx.Lock()
	delete(bk.peers.peers, peerID)
	bk.mtx.Unlock()
	log.WithField("ID", peerID).Info("Delete peer from blockKeeper")
	bk.quitReqBlockCh <- &peerID
}

func (bk *blockKeeper) BroadcastTx(tx *types.Tx) error {
	msg, err := NewTransactionNotifyMessage(tx)
	if err != nil {
		return errors.New("Failed construction tx msg")
	}
	peers := bk.peers.PeersWithoutTx(tx.ID.Byte32())
	for _, peer := range peers {
		bk.peers.peers[peer.Key].MarkTransaction(tx.ID.Byte32())
		peer.Send(BlockchainChannel, struct{ BlockchainMessage }{msg})
	}
	return nil
}

func (bk *blockKeeper) BroadcastMinedBlock(block *types.Block) error {
	msg, err := NewMinedBlockMessage(block)
	if err != nil {
		return errors.New("Failed construction block msg")
	}
	peers := bk.peers.PeersWithoutBlock(block.Hash().Byte32())
	for _, peer := range peers {
		bk.peers.MarkBlock(peer.Key, block.Hash().Byte32())
		peer.Send(BlockchainChannel, struct{ BlockchainMessage }{msg})
	}
	return nil
}

func (bk *blockKeeper) BlockRequestWorker(peerID string, maxPeerHeight uint64) error {
	chainHeight := bk.chain.Height()
	isOrphan := false
	var hash *bc.Hash
	for num := chainHeight + 1; num <= maxPeerHeight; {
		//orphanHeight :=
		block, err := bk.BlockRequest(peerID, num, hash, isOrphan)
		if errors.Root(err) == errPeerDropped || errors.Root(err) == errGetBlockTimeout {
			log.WithField("Peer abnormality. PeerID: ", peerID).Info(err)
			return errCommAbnorm
		}

		isOrphan, err = bk.chain.ProcessBlock(block)
		if err != nil {
			bk.sw.AddScamPeer(bk.peers.Peer(peerID).getPeer())
			log.WithField("hash: ", block.Hash()).Errorf("blockKeeper fail process block %v", err)
			return errScamPeer
		}
		if isOrphan {
			hash = &block.PreviousBlockHash
			continue
		}
		num++
	}
	return nil
}

func (bk *blockKeeper) blockRequest(peerID string, height uint64, hash *bc.Hash, isOrphan bool) {
	if isOrphan == true {
		bk.peers.requestBlockByHash(peerID, hash)
	} else {
		bk.peers.requestBlockByHeight(peerID, height)
	}
}

func (bk *blockKeeper) BlockRequest(peerID string, height uint64, hash *bc.Hash, isOrphan bool) (*types.Block, error) {
	var block *types.Block

	bk.blockRequest(peerID, height, hash, isOrphan)
	retryTicker := time.Tick(requestRetryTicker)
	syncWait := time.NewTimer(syncTimeout)

	for {
		select {
		case pendingResponse := <-bk.pendingProcessCh:
			block = pendingResponse.block
			if strings.Compare(pendingResponse.peerID, peerID) != 0 {
				log.Warning("From different peer")
				continue
			}
			if block.Height != height && isOrphan == false {
				log.Warning("Block height error")
				continue
			}
			tmpHash := block.Hash()
			if strings.Compare(hash.String(), (&tmpHash).String()) != 0 && isOrphan == true {
				log.Warning("Block hash error")
				continue
			}
			return block, nil
		case <-retryTicker:
			bk.blockRequest(peerID, height, hash, isOrphan)
		case <-syncWait.C:
			log.Warning("Request block timeout")
			return nil, errGetBlockTimeout
		case peerid := <-bk.quitReqBlockCh:
			if strings.Compare(*peerid, peerID) == 0 {
				log.Info("Quite block request worker")
				return nil, errPeerDropped
			}
		}
	}
}

func (bk *blockKeeper) txsProcessWorker() {
	for txsResponse := range bk.txsProcessCh {
		tx := txsResponse.tx
		bk.peers.MarkTransaction(txsResponse.peerID, tx.ID.Byte32())
		if err := bk.chain.ValidateTx(tx); err != nil {
			bk.sw.AddScamPeer(bk.peers.Peer(txsResponse.peerID).getPeer())
		}
	}
}
