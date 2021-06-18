package chainmgr

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/consensus"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/p2p/security"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

const (
	syncCycle = 5 * time.Second

	noNeedSync = iota
	fastSyncType
	regularSyncType
)

var (
	maxNumOfBlocksPerMsg      = uint64(64)
	maxNumOfHeadersPerMsg     = uint64(1000)
	maxNumOfBlocksRegularSync = uint64(128)
)

// Fetcher is the interface for fetch struct
type Fetcher interface {
	processBlock(peerID string, block *types.Block)
	processBlocks(peerID string, blocks []*types.Block)
	processHeaders(peerID string, headers []*types.BlockHeader)
	requireBlock(peerID string, height uint64) (*types.Block, error)
}

type blockMsg struct {
	block  *types.Block
	peerID string
}

type blocksMsg struct {
	blocks []*types.Block
	peerID string
}

type headersMsg struct {
	headers []*types.BlockHeader
	peerID  string
}

type blockKeeper struct {
	chain      Chain
	fastSync   *fastSync
	msgFetcher Fetcher
	peers      *peers.PeerSet
	syncPeer   *peers.Peer

	quit chan struct{}
}

func newBlockKeeper(chain Chain, peers *peers.PeerSet, fastSyncDB dbm.DB) *blockKeeper {
	storage := newStorage(fastSyncDB)
	msgFetcher := newMsgFetcher(storage, peers)
	return &blockKeeper{
		chain:      chain,
		fastSync:   newFastSync(chain, msgFetcher, storage, peers),
		msgFetcher: msgFetcher,
		peers:      peers,
		quit:       make(chan struct{}),
	}
}

func (bk *blockKeeper) locateBlocks(locator []*bc.Hash, stopHash *bc.Hash, isTimeout func() bool) ([]*types.Block, error) {
	headers, err := bk.locateHeaders(locator, stopHash, 0, maxNumOfBlocksPerMsg)
	if err != nil {
		return nil, err
	}

	blocks := []*types.Block{}
	for _, header := range headers {
		headerHash := header.Hash()
		block, err := bk.chain.GetBlockByHash(&headerHash)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
		if isTimeout() {
			break
		}
	}
	return blocks, nil
}

func (bk *blockKeeper) locateHeaders(locator []*bc.Hash, stopHash *bc.Hash, skip uint64, maxNum uint64) ([]*types.BlockHeader, error) {
	startHeader, err := bk.chain.GetHeaderByHeight(0)
	if err != nil {
		return nil, err
	}

	for _, hash := range locator {
		header, err := bk.chain.GetHeaderByHash(hash)
		if err == nil && bk.chain.InMainChain(header.Hash()) {
			startHeader = header
			break
		}
	}

	headers := make([]*types.BlockHeader, 0)
	stopHeader, err := bk.chain.GetHeaderByHash(stopHash)
	if err != nil {
		return headers, err
	}

	if !bk.chain.InMainChain(*stopHash) || stopHeader.Height < startHeader.Height {
		return headers, nil
	}

	headers = append(headers, startHeader)
	if stopHeader.Height == startHeader.Height {
		return headers, nil
	}

	for num, index := uint64(0), startHeader.Height; num < maxNum-1; num++ {
		index += skip + 1
		if index >= stopHeader.Height {
			headers = append(headers, stopHeader)
			break
		}

		header, err := bk.chain.GetHeaderByHeight(index)
		if err != nil {
			return nil, err
		}

		headers = append(headers, header)
	}

	return headers, nil
}

func (bk *blockKeeper) processBlock(peerID string, block *types.Block) {
	bk.msgFetcher.processBlock(peerID, block)
}

func (bk *blockKeeper) processBlocks(peerID string, blocks []*types.Block) {
	bk.msgFetcher.processBlocks(peerID, blocks)
}

func (bk *blockKeeper) processHeaders(peerID string, headers []*types.BlockHeader) {
	bk.msgFetcher.processHeaders(peerID, headers)
}

func (bk *blockKeeper) regularBlockSync() error {
	peerHeight := bk.syncPeer.Height()
	bestHeight := bk.chain.BestBlockHeight()
	targetHeight := bestHeight + maxNumOfBlocksRegularSync
	if targetHeight > peerHeight {
		targetHeight = peerHeight
	}

	for i := bestHeight + 1; i <= targetHeight; {
		block, err := bk.msgFetcher.requireBlock(bk.syncPeer.ID(), i)
		if err != nil {
			bk.peers.ProcessIllegal(bk.syncPeer.ID(), security.LevelConnException, err.Error())
			return err
		}

		isOrphan, err := bk.chain.ProcessBlock(block)
		if err != nil {
			bk.peers.ProcessIllegal(bk.syncPeer.ID(), security.LevelMsgIllegal, err.Error())
			return err
		}

		if isOrphan {
			i--
			continue
		}

		//This code is used to preventing the sync peer return a dust block which will not change the node's chain status
		if bestHeight = bk.chain.BestBlockHeight(); i == bestHeight+1 {
			log.WithFields(log.Fields{"module": logModule, "height": i}).Warn("stop regular sync due to loop sync same height")
			return nil
		}

		i = bestHeight + 1
	}
	log.WithFields(log.Fields{"module": logModule, "height": bk.chain.BestBlockHeight()}).Info("regular sync success")
	return nil
}

func (bk *blockKeeper) start() {
	go bk.syncWorker()
}

func (bk *blockKeeper) checkSyncType() int {
	bestHeight := bk.chain.BestBlockHeight()
	peer := bk.peers.BestPeer(consensus.SFFullNode | consensus.SFFastSync)
	if peer != nil {
		if peerJustifiedHeight := peer.JustifiedHeight(); peerJustifiedHeight >= bestHeight+minGapStartFastSync {
			bk.fastSync.setSyncPeer(peer)
			return fastSyncType
		}
	}

	peer = bk.peers.BestPeer(consensus.SFFullNode)
	if peer == nil {
		log.WithFields(log.Fields{"module": logModule}).Debug("can't find sync peer")
		return noNeedSync
	}

	if peer.Height() > bestHeight {
		bk.syncPeer = peer
		return regularSyncType
	}

	return noNeedSync
}

func (bk *blockKeeper) startSync() bool {
	switch bk.checkSyncType() {
	case fastSyncType:
		if err := bk.fastSync.process(); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("failed on fast sync")
			return false
		}
	case regularSyncType:
		if err := bk.regularBlockSync(); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on regularBlockSync")
			return false
		}
	default:
		return false
	}

	return true
}

func (bk *blockKeeper) stop() {
	close(bk.quit)
}

func (bk *blockKeeper) syncWorker() {
	syncTicker := time.NewTicker(syncCycle)
	defer syncTicker.Stop()

	for {
		select {
		case <-syncTicker.C:
			if update := bk.startSync(); !update {
				continue
			}

			lastJustifiedHeader, err := bk.chain.LastJustifiedHeader()
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail get last just justified header")
			}

			if err := bk.peers.BroadcastNewStatus(bk.chain.BestBlockHeader(), lastJustifiedHeader); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on syncWorker broadcast new status")
			}
		case <-bk.quit:
			return
		}
	}
}
