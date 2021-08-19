package chainmgr

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/p2p/security"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

var (
	minSizeOfSyncSkeleton  = 2
	maxSizeOfSyncSkeleton  = 11
	numOfBlocksSkeletonGap = maxNumOfBlocksPerMsg
	maxNumOfBlocksPerSync  = numOfBlocksSkeletonGap * uint64(maxSizeOfSyncSkeleton-1)
	fastSyncPivotGap       = uint64(64)
	minGapStartFastSync    = uint64(128)

	errNoSyncPeer      = errors.New("can't find sync peer")
	errSkeletonSize    = errors.New("fast sync skeleton size wrong")
	errNoMainSkeleton  = errors.New("No main skeleton found")
	errNoSkeletonFound = errors.New("No skeleton found")
)

type fastSync struct {
	chain          Chain
	msgFetcher     MsgFetcher
	blockProcessor *blockProcessor
	peers          *peers.PeerSet
	mainSyncPeer   *peers.Peer
}

func newFastSync(chain Chain, msgFetcher MsgFetcher, storage *storage, peers *peers.PeerSet) *fastSync {
	return &fastSync{
		chain:          chain,
		msgFetcher:     msgFetcher,
		blockProcessor: newBlockProcessor(chain, storage, peers),
		peers:          peers,
	}
}

func (fs *fastSync) blockLocator() []*bc.Hash {
	header := fs.chain.BestBlockHeader()
	locator := []*bc.Hash{}
	step := uint64(1)

	for header != nil {
		headerHash := header.Hash()
		locator = append(locator, &headerHash)
		if header.Height == 0 {
			break
		}

		var err error
		if header.Height < step {
			header, err = fs.chain.GetHeaderByHeight(0)
		} else {
			header, err = fs.chain.GetHeaderByHeight(header.Height - step)
		}
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("blockKeeper fail on get blockLocator")
			break
		}

		if len(locator) >= 9 {
			step *= 2
		}
	}
	return locator
}

// createFetchBlocksTasks get the skeleton and assign tasks according to the skeleton.
func (fs *fastSync) createFetchBlocksTasks(stopBlock *types.Block) ([]*fetchBlocksWork, error) {
	// Find peers that meet the height requirements.
	peers := fs.peers.GetPeersByHeight(stopBlock.Height + fastSyncPivotGap)
	if len(peers) == 0 {
		return nil, errNoSyncPeer
	}

	// parallel fetch the skeleton from peers.
	stopHash := stopBlock.Hash()
	skeletonMap := fs.msgFetcher.parallelFetchHeaders(peers, fs.blockLocator(), &stopHash, numOfBlocksSkeletonGap-1)
	if len(skeletonMap) == 0 {
		return nil, errNoSkeletonFound
	}

	mainSkeleton, ok := skeletonMap[fs.mainSyncPeer.ID()]
	if !ok {
		return nil, errNoMainSkeleton
	}

	if len(mainSkeleton) < minSizeOfSyncSkeleton {
		fs.peers.ProcessIllegal(fs.mainSyncPeer.ID(), security.LevelMsgIllegal, errSkeletonSize.Error())
		return nil, errSkeletonSize
	}

	// collect peers that match the skeleton of the primary sync peer
	fs.msgFetcher.addSyncPeer(fs.mainSyncPeer.ID())
	delete(skeletonMap, fs.mainSyncPeer.ID())
	for peerID, skeleton := range skeletonMap {
		if len(skeleton) != len(mainSkeleton) {
			log.WithFields(log.Fields{"module": logModule, "main skeleton": len(mainSkeleton), "got skeleton": len(skeleton)}).Warn("different skeleton length")
			continue
		}

		for i, header := range skeleton {
			if header.Hash() != mainSkeleton[i].Hash() {
				log.WithFields(log.Fields{"module": logModule, "header index": i, "main skeleton": mainSkeleton[i].Hash(), "got skeleton": header.Hash()}).Warn("different skeleton hash")
				continue
			}
		}
		fs.msgFetcher.addSyncPeer(peerID)
	}

	blockFetchTasks := make([]*fetchBlocksWork, 0)
	// create download task
	for i := 0; i < len(mainSkeleton)-1 && i < maxSizeOfSyncSkeleton-1; i++ {
		blockFetchTasks = append(blockFetchTasks, &fetchBlocksWork{startHeader: mainSkeleton[i], stopHeader: mainSkeleton[i+1]})
	}

	return blockFetchTasks, nil
}

func (fs *fastSync) process() error {
	stopBlock, err := fs.findSyncRange()
	if err != nil {
		return err
	}

	tasks, err := fs.createFetchBlocksTasks(stopBlock)
	if err != nil {
		return err
	}

	downloadNotifyCh := make(chan struct{}, 1)
	processStopCh := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go fs.msgFetcher.parallelFetchBlocks(tasks, downloadNotifyCh, processStopCh, &wg)
	go fs.blockProcessor.process(downloadNotifyCh, processStopCh, tasks[0].startHeader.Height, &wg)
	wg.Wait()
	fs.msgFetcher.resetParameter()
	log.WithFields(log.Fields{"module": logModule, "height": fs.chain.BestBlockHeight()}).Info("fast sync complete")
	return nil
}

// findSyncRange find the start and end of this sync.
// sync length cannot be greater than maxFastSyncBlocksNum.
func (fs *fastSync) findSyncRange() (*types.Block, error) {
	bestHeight := fs.chain.BestBlockHeight()
	length := fs.mainSyncPeer.Height() - fastSyncPivotGap - bestHeight
	if length > maxNumOfBlocksPerSync {
		length = maxNumOfBlocksPerSync
	}

	return fs.msgFetcher.requireBlock(fs.mainSyncPeer.ID(), bestHeight+length)
}

func (fs *fastSync) setSyncPeer(peer *peers.Peer) {
	fs.mainSyncPeer = peer
}
