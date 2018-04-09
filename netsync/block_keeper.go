package netsync

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
	"github.com/bytom/p2p"
	"github.com/bytom/protocol"
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
	errReqBlock        = errors.New("Request block error")
)

//TODO: add retry mechanism
type blockKeeper struct {
	chain *protocol.Chain
	sw    *p2p.Switch
	peers *peerSet

	pendingProcessCh chan *blockPending
	txsProcessCh     chan *txsNotify
	quitReqBlockCh   chan *string
}

func newBlockKeeper(chain *protocol.Chain, sw *p2p.Switch, peers *peerSet, quitReqBlockCh chan *string) *blockKeeper {
	bk := &blockKeeper{
		chain:            chain,
		sw:               sw,
		peers:            peers,
		pendingProcessCh: make(chan *blockPending, maxBlocksPending),
		txsProcessCh:     make(chan *txsNotify, maxtxsPending),
		quitReqBlockCh:   quitReqBlockCh,
	}
	go bk.txsProcessWorker()
	return bk
}

func (bk *blockKeeper) AddBlock(block *types.Block, peerID string) {
	bk.pendingProcessCh <- &blockPending{block: block, peerID: peerID}
}

func (bk *blockKeeper) AddTx(tx *types.Tx, peerID string) {
	bk.txsProcessCh <- &txsNotify{tx: tx, peerID: peerID}
}

func (bk *blockKeeper) IsCaughtUp() bool {
	_, height := bk.peers.BestPeer()
	return bk.chain.Height() < height
}

func (bk *blockKeeper) BlockRequestWorker(peerID string, maxPeerHeight uint64) error {
	num := bk.chain.Height() + 1
	orphanNum := uint64(0)
	reqNum := uint64(0)
	isOrphan := false
	for num <= maxPeerHeight && num > 0 {
		if isOrphan {
			reqNum = orphanNum
		} else {
			reqNum = num
		}
		block, err := bk.BlockRequest(peerID, reqNum)
		if errors.Root(err) == errPeerDropped || errors.Root(err) == errGetBlockTimeout || errors.Root(err) == errReqBlock {
			log.WithField("Peer abnormality. PeerID: ", peerID).Info(err)
			peer, ok := bk.peers.peers[peerID]
			if !ok {
				return errNotRegistered
			}
			bk.sw.StopPeerGracefully(peer.Peer)
			return errCommAbnorm
		}
		isOrphan, err = bk.chain.ProcessBlock(block)
		if err != nil {
			bk.sw.AddScamPeer(bk.peers.Peer(peerID).getPeer())
			log.WithField("hash: ", block.Hash()).Errorf("blockKeeper fail process block %v", err)
			return errScamPeer
		}
		if isOrphan {
			orphanNum = block.Height - 1
			continue
		}
		num++
	}
	return nil
}

func (bk *blockKeeper) blockRequest(peerID string, height uint64) error {
	return bk.peers.requestBlockByHeight(peerID, height)
}

func (bk *blockKeeper) BlockRequest(peerID string, height uint64) (*types.Block, error) {
	var block *types.Block

	if err := bk.blockRequest(peerID, height); err != nil {
		return nil, errReqBlock
	}
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
			if block.Height != height {
				log.Warning("Block height error")
				continue
			}
			return block, nil
		case <-retryTicker:
			if err := bk.blockRequest(peerID, height); err != nil {
				return nil, errReqBlock
			}
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
		log.Info("Receive new tx from remote peer. TxID:", tx.ID.String())
		bk.peers.MarkTransaction(txsResponse.peerID, &tx.ID)
		if isOrphan, err := bk.chain.ValidateTx(tx); err != nil && isOrphan == false {
			bk.sw.AddScamPeer(bk.peers.Peer(txsResponse.peerID).getPeer())
		}
	}
}
