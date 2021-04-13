package chainmgr

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/p2p/security"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

const (
	maxNumOfParallelFetchBlocks = 7
	blockProcessChSize          = 1024
	blocksProcessChSize         = 128
	headersProcessChSize        = 1024
	maxNumOfFastSyncPeers       = 128
)

var (
	requireBlockTimeout      = 20 * time.Second
	requireHeadersTimeout    = 30 * time.Second
	requireBlocksTimeout     = 90 * time.Second
	checkSyncPeerNumInterval = 5 * time.Second

	errRequestBlocksTimeout = errors.New("request blocks timeout")
	errRequestTimeout       = errors.New("request timeout")
	errPeerDropped          = errors.New("Peer dropped")
	errSendMsg              = errors.New("send message error")
)

// MsgFetcher is the interface for msg fetch struct
type MsgFetcher interface {
	resetParameter()
	addSyncPeer(peerID string)
	requireBlock(peerID string, height uint64) (*types.Block, error)
	parallelFetchBlocks(work []*fetchBlocksWork, downloadNotifyCh chan struct{}, ProcessStopCh chan struct{}, wg *sync.WaitGroup)
	parallelFetchHeaders(peers []*peers.Peer, locator []*bc.Hash, stopHash *bc.Hash, skip uint64) map[string][]*types.BlockHeader
}

type fetchBlocksWork struct {
	startHeader, stopHeader *types.BlockHeader
}

type fetchBlocksResult struct {
	startHeight, stopHeight uint64
	err                     error
}

type msgFetcher struct {
	storage          *storage
	syncPeers        *fastSyncPeers
	peers            *peers.PeerSet
	blockProcessCh   chan *blockMsg
	blocksProcessCh  chan *blocksMsg
	headersProcessCh chan *headersMsg
	blocksMsgChanMap map[string]chan []*types.Block
	mux              sync.RWMutex
}

func newMsgFetcher(storage *storage, peers *peers.PeerSet) *msgFetcher {
	return &msgFetcher{
		storage:          storage,
		syncPeers:        newFastSyncPeers(),
		peers:            peers,
		blockProcessCh:   make(chan *blockMsg, blockProcessChSize),
		blocksProcessCh:  make(chan *blocksMsg, blocksProcessChSize),
		headersProcessCh: make(chan *headersMsg, headersProcessChSize),
		blocksMsgChanMap: make(map[string]chan []*types.Block),
	}
}

func (mf *msgFetcher) addSyncPeer(peerID string) {
	mf.syncPeers.add(peerID)
}

func (mf *msgFetcher) collectResultLoop(peerCh chan string, quit chan struct{}, resultCh chan *fetchBlocksResult, workerCloseCh chan struct{}, workSize int) {
	defer close(workerCloseCh)
	ticker := time.NewTicker(checkSyncPeerNumInterval)
	defer ticker.Stop()

	//collect fetch results
	for resultCount := 0; resultCount < workSize && mf.syncPeers.size() > 0; {
		select {
		case result := <-resultCh:
			resultCount++
			if result.err != nil {
				log.WithFields(log.Fields{"module": logModule, "startHeight": result.startHeight, "stopHeight": result.stopHeight, "err": result.err}).Error("failed on fetch blocks")
				return
			}

			peer, err := mf.syncPeers.selectIdlePeer()
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": result.err}).Warn("failed on find fast sync peer")
				break
			}
			peerCh <- peer
		case <-ticker.C:
			if mf.syncPeers.size() == 0 {
				log.WithFields(log.Fields{"module": logModule}).Warn("num of fast sync peer is 0")
				return
			}
		case _, ok := <-quit:
			if !ok {
				return
			}
		}
	}
}

func (mf *msgFetcher) fetchBlocks(work *fetchBlocksWork, peerID string) ([]*types.Block, error) {
	defer mf.syncPeers.setIdle(peerID)
	startHash := work.startHeader.Hash()
	stopHash := work.stopHeader.Hash()
	blocks, err := mf.requireBlocks(peerID, []*bc.Hash{&startHash}, &stopHash)
	if err != nil {
		mf.syncPeers.delete(peerID)
		mf.peers.ProcessIllegal(peerID, security.LevelConnException, err.Error())
		return nil, err
	}

	if err := mf.verifyBlocksMsg(blocks, work.startHeader, work.stopHeader); err != nil {
		mf.syncPeers.delete(peerID)
		mf.peers.ProcessIllegal(peerID, security.LevelConnException, err.Error())
		return nil, err
	}

	return blocks, nil
}

func (mf *msgFetcher) fetchBlocksProcess(work *fetchBlocksWork, peerCh chan string, downloadNotifyCh chan struct{}, closeCh chan struct{}) error {
	for {
		select {
		case peerID := <-peerCh:
			for {
				blocks, err := mf.fetchBlocks(work, peerID)
				if err != nil {
					log.WithFields(log.Fields{"module": logModule, "startHeight": work.startHeader.Height, "stopHeight": work.stopHeader.Height, "error": err}).Info("failed on fetch blocks")
					break
				}

				if err := mf.storage.writeBlocks(peerID, blocks); err != nil {
					log.WithFields(log.Fields{"module": logModule, "error": err}).Info("write block error")
					return err
				}

				// send to block process pool
				select {
				case downloadNotifyCh <- struct{}{}:
				default:
				}

				// work completed
				if blocks[len(blocks)-1].Height >= work.stopHeader.Height-1 {
					return nil
				}

				//unfinished work, continue
				work.startHeader = &blocks[len(blocks)-1].BlockHeader
			}
		case <-closeCh:
			return nil
		}
	}
}

func (mf *msgFetcher) fetchBlocksWorker(workCh chan *fetchBlocksWork, peerCh chan string, resultCh chan *fetchBlocksResult, closeCh chan struct{}, downloadNotifyCh chan struct{}, wg *sync.WaitGroup) {
	for {
		select {
		case work := <-workCh:
			err := mf.fetchBlocksProcess(work, peerCh, downloadNotifyCh, closeCh)
			resultCh <- &fetchBlocksResult{startHeight: work.startHeader.Height, stopHeight: work.stopHeader.Height, err: err}
		case <-closeCh:
			wg.Done()
			return
		}
	}
}

func (mf *msgFetcher) parallelFetchBlocks(works []*fetchBlocksWork, downloadNotifyCh chan struct{}, ProcessStopCh chan struct{}, wg *sync.WaitGroup) {
	workSize := len(works)
	workCh := make(chan *fetchBlocksWork, workSize)
	peerCh := make(chan string, maxNumOfFastSyncPeers)
	resultCh := make(chan *fetchBlocksResult, workSize)
	closeCh := make(chan struct{})

	for _, work := range works {
		workCh <- work
	}
	syncPeers := mf.syncPeers.selectIdlePeers()
	for i := 0; i < len(syncPeers) && i < maxNumOfFastSyncPeers; i++ {
		peerCh <- syncPeers[i]
	}

	var workWg sync.WaitGroup
	for i := 0; i <= maxNumOfParallelFetchBlocks && i < workSize; i++ {
		workWg.Add(1)
		go mf.fetchBlocksWorker(workCh, peerCh, resultCh, closeCh, downloadNotifyCh, &workWg)
	}

	go mf.collectResultLoop(peerCh, ProcessStopCh, resultCh, closeCh, workSize)

	workWg.Wait()
	close(resultCh)
	close(peerCh)
	close(workCh)
	close(downloadNotifyCh)
	wg.Done()
}

func (mf *msgFetcher) parallelFetchHeaders(peers []*peers.Peer, locator []*bc.Hash, stopHash *bc.Hash, skip uint64) map[string][]*types.BlockHeader {
	result := make(map[string][]*types.BlockHeader)
	response := make(map[string]bool)
	for _, peer := range peers {
		if ok := peer.GetHeaders(locator, stopHash, skip); !ok {
			continue
		}
		result[peer.ID()] = nil
	}

	timeout := time.NewTimer(requireHeadersTimeout)
	defer timeout.Stop()
	for {
		select {
		case msg := <-mf.headersProcessCh:
			if _, ok := result[msg.peerID]; ok {
				result[msg.peerID] = append(result[msg.peerID], msg.headers[:]...)
				response[msg.peerID] = true
				if len(response) == len(result) {
					return result
				}
			}
		case <-timeout.C:
			log.WithFields(log.Fields{"module": logModule, "err": errRequestTimeout}).Warn("failed on parallel fetch headers")
			return result
		}
	}
}

func (mf *msgFetcher) processBlock(peerID string, block *types.Block) {
	mf.blockProcessCh <- &blockMsg{block: block, peerID: peerID}
}

func (mf *msgFetcher) processBlocks(peerID string, blocks []*types.Block) {
	mf.blocksProcessCh <- &blocksMsg{blocks: blocks, peerID: peerID}
	mf.mux.RLock()
	blocksMsgChan, ok := mf.blocksMsgChanMap[peerID]
	mf.mux.RUnlock()
	if !ok {
		mf.peers.ProcessIllegal(peerID, security.LevelMsgIllegal, "msg from unsolicited peer")
		return
	}

	blocksMsgChan <- blocks
}

func (mf *msgFetcher) processHeaders(peerID string, headers []*types.BlockHeader) {
	mf.headersProcessCh <- &headersMsg{headers: headers, peerID: peerID}
}

func (mf *msgFetcher) requireBlock(peerID string, height uint64) (*types.Block, error) {
	peer := mf.peers.GetPeer(peerID)
	if peer == nil {
		return nil, errPeerDropped
	}

	if ok := peer.GetBlockByHeight(height); !ok {
		return nil, errSendMsg
	}

	timeout := time.NewTimer(requireBlockTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-mf.blockProcessCh:
			if msg.peerID != peerID {
				continue
			}
			if msg.block.Height != height {
				continue
			}
			return msg.block, nil
		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireBlock")
		}
	}
}

func (mf *msgFetcher) requireBlocks(peerID string, locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error) {
	peer := mf.peers.GetPeer(peerID)
	if peer == nil {
		mf.syncPeers.delete(peerID)
		return nil, errPeerDropped
	}

	receiveCh := make(chan []*types.Block, 1)
	mf.mux.Lock()
	mf.blocksMsgChanMap[peerID] = receiveCh
	mf.mux.Unlock()

	if ok := peer.GetBlocks(locator, stopHash); !ok {
		return nil, errSendMsg
	}

	timeout := time.NewTimer(requireBlocksTimeout)
	defer timeout.Stop()
	select {
	case blocks := <-receiveCh:
		return blocks, nil
	case <-timeout.C:
		return nil, errRequestBlocksTimeout
	}
}

func (mf *msgFetcher) resetParameter() {
	mf.blocksMsgChanMap = make(map[string]chan []*types.Block)
	mf.syncPeers = newFastSyncPeers()
	mf.storage.resetParameter()
	//empty chan
	for {
		select {
		case <-mf.blocksProcessCh:
		case <-mf.headersProcessCh:
		default:
			return
		}
	}
}

func (mf *msgFetcher) verifyBlocksMsg(blocks []*types.Block, startHeader, stopHeader *types.BlockHeader) error {
	// null blocks
	if len(blocks) == 0 {
		return errors.New("null blocks msg")
	}

	// blocks more than request
	if uint64(len(blocks)) > stopHeader.Height-startHeader.Height+1 {
		return errors.New("exceed length blocks msg")
	}

	// verify start block
	if blocks[0].Hash() != startHeader.Hash() {
		return errors.New("get mismatch blocks msg")
	}

	// verify blocks continuity
	for i := 0; i < len(blocks)-1; i++ {
		if blocks[i].Hash() != blocks[i+1].PreviousBlockHash {
			return errors.New("get discontinuous blocks msg")
		}
	}

	return nil
}
