package netsync

import (
	"container/list"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/errors"
	"github.com/bytom/mining/tensority"
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

	maxBlocksPending  = 1024
	maxtxsPending     = 32768
	maxHeadersPending = 32

	maxQuitReq = 256

	maxTxChanSize          = 10000 // txChanSize is the size of channel listening to Txpool newTxCh
	MaxRequestBlocksPerMsg = 20
)

var (
	errGetBlockTimeout  = errors.New("Get block Timeout")
	errGetBlocksTimeout = errors.New("Get blocks Timeout")
	errPeerDropped      = errors.New("Peer dropped")
	errGetBlockByHash   = errors.New("Get block by hash error")
	errBroadcastStatus  = errors.New("Broadcast new status block error")
	errReqBlock         = errors.New("Request block error")
	errReqHeaders       = errors.New("Request block headers error")
	errPeerNotRegister  = errors.New("peer is not registered")
	errPeerMisbehave    = errors.New("peer is misbehave")
	errEmptyHeaders     = errors.New("headers is empty")
)

//TODO: add retry mechanism
type blockKeeper struct {
	chain *protocol.Chain
	sw    *p2p.Switch
	peers *peerSet

	pendingProcessCh chan *blockPending
	headersProcessCh chan *headersMsg

	txsProcessCh   chan *txsNotify
	quitReqBlockCh chan *string
	headerList     *list.List
	startHeader    *list.Element
}

func newBlockKeeper(chain *protocol.Chain, sw *p2p.Switch, peers *peerSet, quitReqBlockCh chan *string) *blockKeeper {
	best := chain.BestBlockHeader()
	bestHash := best.Hash()
	bk := &blockKeeper{
		chain:            chain,
		sw:               sw,
		peers:            peers,
		pendingProcessCh: make(chan *blockPending, maxBlocksPending),
		txsProcessCh:     make(chan *txsNotify, maxtxsPending),
		headersProcessCh: make(chan *headersMsg, maxHeadersPending),
		quitReqBlockCh:   quitReqBlockCh,
	}
	if nextCheckPoint := bk.nextCheckpoint(); nextCheckPoint != nil {
		bk.resetHeaderState(common.CoreHashToHash(&bestHash), best.Height)
	}
	go bk.txsProcessWorker()
	return bk
}

// resetHeaderState sets the headers-first mode state to values appropriate for
// syncing from a new peer.
func (bk *blockKeeper) resetHeaderState(newestHash *common.Hash, newestHeight uint64) {
	bk.headerList = list.New()
	bk.startHeader = nil

	// When there is a next checkpoint, add an entry for the latest known
	// block into the header pool.  This allows the next downloaded header
	// to prove it links to the chain properly.
	if bk.nextCheckpoint() != nil {
		node := headerNode{height: newestHeight, hash: newestHash}
		bk.headerList.PushBack(&node)
	}
}

func (bk *blockKeeper) AddBlock(block *types.Block, peerID string) {
	bk.pendingProcessCh <- &blockPending{block: block, peerID: peerID}
}

func (bk *blockKeeper) AddTx(tx *types.Tx, peerID string) {
	bk.txsProcessCh <- &txsNotify{tx: tx, peerID: peerID}
}

func (bk *blockKeeper) IsCaughtUp() bool {
	_, height := bk.peers.BestPeer()
	return bk.chain.BestBlockHeight() < height
}

func (bk *blockKeeper) BlockRequestWorker(peerID string, maxPeerHeight uint64) error {
	num := bk.chain.BestBlockHeight() + 1
	currentHash := bk.chain.BestBlockHash()
	orphanNum := uint64(0)
	reqNum := uint64(0)
	isOrphan := false
	bkPeer, ok := bk.peers.Peer(peerID)
	if !ok {
		log.Info("peer is not registered")
		return errPeerNotRegister
	}
	swPeer := bkPeer.getPeer()
	for 0 < num && num <= maxPeerHeight {
		if nextCheckPoint := bk.nextCheckpoint(); nextCheckPoint != nil {
			_, bestHeight := bk.peers.BestPeer()
			if bestHeight > nextCheckPoint.Height {
				log.Info("Switch to fast sync mode")
				return nil
			}
		}
		if isOrphan {
			reqNum = orphanNum
		} else {
			reqNum = num
		}
		block, err := bk.BlockRequest(peerID, reqNum)
		if errors.Root(err) == errPeerDropped || errors.Root(err) == errGetBlockTimeout || errors.Root(err) == errReqBlock {
			log.WithField("Peer abnormality. PeerID: ", peerID).Info(err)
			if bkPeer == nil {
				log.Info("peer is not registered")
				break
			}
			log.Info("Block keeper request block error. Stop peer.")
			bk.sw.StopPeerGracefully(swPeer)
			break
		}
		isOrphan, err = bk.chain.ProcessBlock(block)
		if err != nil {
			if bkPeer == nil {
				log.Info("peer is deleted")
				break
			}
			if ban := bkPeer.addBanScore(20, 0, "block process error"); ban {
				bk.sw.AddBannedPeer(swPeer)
				bk.sw.StopPeerGracefully(swPeer)
			}
			log.WithField("hash:", block.Hash()).Errorf("blockKeeper fail process block %v ", err)
			break
		}
		if isOrphan {
			orphanNum = block.Height - 1
			continue
		}
		num++
	}
	bestHash := bk.chain.BestBlockHash()
	log.Info("Block sync complete. height:", bk.chain.BestBlockHeight(), " hash:", bestHash)
	if currentHash.String() != bestHash.String() {
		log.Info("Broadcast new chain status.")

		block, err := bk.chain.GetBlockByHash(bestHash)
		if err != nil {
			log.Errorf("Failed on sync complete broadcast status get block %v", err)
			return errGetBlockByHash
		}

		peers, err := bk.peers.BroadcastNewStatus(block)
		if err != nil {
			log.Errorf("Failed on broadcast new status block %v", err)
			return errBroadcastStatus
		}
		for _, peer := range peers {
			if peer == nil {
				return errPeerNotRegister
			}
			swPeer := peer.getPeer()
			log.Info("Block keeper broadcast block error. Stop peer.")
			bk.sw.StopPeerGracefully(swPeer)
		}
	}
	return nil
}

func (bk *blockKeeper) HeadersRequest(peerID string, locator []*common.Hash) ([]types.BlockHeader, error) {
	nextCheckPoint := bk.nextCheckpoint()
	stopHash := common.CoreHashToHash(&nextCheckPoint.Hash)
	if err := bk.getHeaders(peerID, locator, stopHash); err != nil {
		log.Info("getHeaders err")
		return nil, errReqHeaders
	}
	retryTicker := time.Tick(requestRetryTicker)
	syncWait := time.NewTimer(syncTimeout)
	var headers []types.BlockHeader

	for {
		select {
		case pendingResponse := <-bk.headersProcessCh:
			headers = pendingResponse.headers
			if pendingResponse.peerID != peerID {
				log.Warning("From different peer")
				continue
			}

			return headers, nil
		case <-retryTicker:
			if err := bk.getHeaders(peerID, locator, stopHash); err != nil {
				return nil, errReqHeaders
			}
		case <-syncWait.C:
			log.Warning("Request block timeout")
			return nil, errGetBlockTimeout
		case peerid := <-bk.quitReqBlockCh:
			if *peerid == peerID {
				log.Info("Quite block headers request worker")
				return nil, errPeerDropped
			}
		}
	}
}

func (bk *blockKeeper) BlockFastSyncWorker() error {
	//request blocks header
	bestPeer, bestHeight := bk.peers.BestPeer()
	nextCheckPoint := bk.nextCheckpoint()
	totalHeaders := make([]types.BlockHeader, 0)

	if bestHeight > nextCheckPoint.Height {
		locator := bk.blockLocator(nil)
		for {
			headers, err := bk.HeadersRequest(bestPeer.Key, locator)
			if err != nil {
				log.Info("HeadersRequest err")
				return err
			}
			err, receivedCheckpoint := bk.handleHeadersMsg(bestPeer.Key, headers)
			if err != nil {
				log.Info("handleHeadersMsg err")
				return err
			}
			totalHeaders = append(totalHeaders, headers...)
			if receivedCheckpoint {
				break
			}
			finalHash := headers[len(headers)-1].Hash()
			locator = []*common.Hash{common.CoreHashToHash(&finalHash)}
		}
		log.Infof("Downloading headers for blocks %d to "+
			"%d from peer %s", bk.chain.BestBlockHeight()+1,
			bk.nextCheckpoint().Height, bestPeer.Key)
	}

	for e := bk.headerList.Front(); e != nil; {
		headerList := list.New()
		for num := 0; num < MaxRequestBlocksPerMsg; num++ {
			if e == nil {
				break
			}
			headerList.PushBack(e)
			e = e.Next()
		}
		blocks, err := bk.BlocksRequestWorker(bestPeer.Key, headerList, headerList.Len())
		if err != nil {
			return err
		}
		for _, block := range blocks {
			seed, _ := bk.chain.CalcNextSeed(&(block.PreviousBlockHash))
			blockHash := block.Hash()
			tensority.AIHash.AddCache(&blockHash, seed, &bc.Hash{})
			isOrphan, err := bk.chain.ProcessBlock(block)
			if err != nil {
				log.WithField("hash:", block.Hash()).Errorf("blockKeeper fail process block %v ", err)
				return err
			}

			if isOrphan {
				return errors.New("block order error")
			}
		}
	}

	height := bk.chain.BestBlockHeight()
	block, _ := bk.chain.GetBlockByHeight(height)
	peers, err := bk.peers.BroadcastNewStatus(block)
	if err != nil {
		log.Errorf("Failed on broadcast new status block %v", err)
		return errBroadcastStatus
	}
	for _, peer := range peers {
		if peer == nil {
			return errPeerNotRegister
		}
		swPeer := peer.getPeer()
		log.Info("Block keeper broadcast block error. Stop peer.")
		bk.sw.StopPeerGracefully(swPeer)
	}

	return nil
}

func blocksCollect(headerList *list.List, beginHeight uint64, num int, blocks []blockMsg, totalBlocks *[]*types.Block) (bool, error) {
	if len(blocks) > num {
		return false, errors.New("blocks length error")
	}

	for i := 0; i < len(blocks); i++ {
		for e := headerList.Front(); e != nil; e = e.Next() {
			if blocks[i].Hash.Str() == e.Value.(*headerNode).hash.Str() {
				block := &types.Block{
					BlockHeader:  types.BlockHeader{},
					Transactions: []*types.Tx{},
				}
				block.UnmarshalText(blocks[i].RawBlock)
				//todo: add txs merkle check
				(*totalBlocks)[e.Value.(*headerNode).height-beginHeight] = block
				headerList.Remove(e)
				break
			}
		}
	}

	if headerList.Len() == 0 {
		return true, nil
	}

	return false, nil
}

func (bk *blockKeeper) BlocksRequestWorker(peerID string, headerList *list.List, num int) ([]*types.Block, error) {
	peer, _ := bk.peers.Peer(peerID)
	if peer == nil {
		return nil, errPeerDropped
	}

	beginHash := headerList.Front().Value.(*headerNode).hash
	beginHeight := headerList.Front().Value.(*headerNode).height

	if err := bk.peers.requestBlocksByHash(peerID, beginHash, num); err != nil {
		return nil, err
	}
	retryTicker := time.Tick(requestRetryTicker)
	syncWait := time.NewTimer(syncTimeout)
	totalBlocks := make([]*types.Block, num)

	for {
		select {
		case pendingResponse := <-peer.blocksProcessCh:
			blocks := pendingResponse.blocks
			ok, err := blocksCollect(headerList, beginHeight, num, blocks, &totalBlocks)
			if err != nil {
				return nil, err
			}
			if ok {
				return totalBlocks, nil
			}
		case <-retryTicker:
			if err := bk.peers.requestBlocksByHash(peerID, beginHash, num); err != nil {
				return nil, err
			}
		case <-syncWait.C:
			log.Warning("Request blocks timeout")
			return nil, errGetBlocksTimeout
		case peerid := <-bk.quitReqBlockCh:
			if *peerid == peerID {
				log.Info("Quite blocks request worker")
				return nil, errPeerDropped
			}
		}
	}
}

func (bk *blockKeeper) blockRequest(peerID string, height uint64) error {
	return bk.peers.requestBlockByHeight(peerID, height)
}

func (bk *blockKeeper) nextCheckpoint() *consensus.Checkpoint {
	height := bk.chain.BestBlockHeader().Height
	checkpoints := consensus.ActiveNetParams.Checkpoints
	if len(checkpoints) == 0 || height >= checkpoints[len(checkpoints)-1].Height {
		return nil
	}

	nextCheckpoint := &checkpoints[len(checkpoints)-1]
	for i := len(checkpoints) - 2; i >= 0; i-- {
		if height >= checkpoints[i].Height {
			break
		}
		nextCheckpoint = &checkpoints[i]
	}
	return nextCheckpoint
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
			if pendingResponse.peerID != peerID {
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
			if *peerid == peerID {
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
			if bkPeer, ok := bk.peers.Peer(txsResponse.peerID); ok {
				swPeer := bkPeer.getPeer()
				if ban := bkPeer.addBanScore(10, 0, "tx error"); ban {
					bk.sw.AddBannedPeer(swPeer)
					bk.sw.StopPeerGracefully(swPeer)
				}
			}
		}
	}
}

func (bk *blockKeeper) headersSend(peerID string, headers []types.BlockHeader) error {
	return bk.peers.SendHeaders(peerID, headers)
}

// OnGetHeaders is invoked when a peer receives a headersMessage message.
func (bk *blockKeeper) GetHeadersWorker(peerID string, msg *GetHeadersMessage) {
	// Ignore getheaders requests if not in sync.
	//if !sp.server.syncManager.IsCurrent() {
	//	return
	//}

	// Find the most recent known block in the best chain based on the block
	// locator and fetch all of the headers after it until either
	// wire.MaxBlockHeadersPerMsg have been fetched or the provided stop
	// hash is encountered.
	//
	// Use the block after the genesis block if no other blocks in the
	// provided locator are known.  This does mean the client will start
	// over with the genesis block if unknown block locators are provided.
	//
	// This mirrors the behavior in the reference implementation.
	for i := range msg.BlockLocatorHashes {
		log.Info("num:", i, "BlockLocatorHashes:", &msg.BlockLocatorHashes[i])
	}
	log.Info("HashStop:", msg.HashStop)
	headers := bk.locateHeaders(msg.BlockLocatorHashes, &msg.HashStop, MaxBlockHeadersPerMsg)
	if len(headers) == 0 {
		// Nothing to send.
		log.Info("Nothing to send.")
		return
	}
	bk.headersSend(peerID, headers)
}

// LocateHeaders returns the headers of the blocks after the first known block
// in the locator until the provided stop hash is reached, or up to a max of
// wire.MaxBlockHeadersPerMsg headers.
//
// In addition, there are two special cases:
//
// - When no locators are provided, the stop hash is treated as a request for
//   that header, so it will either return the header for the stop hash itself
//   if it is known, or nil if it is unknown
// - When locators are provided, but none of them are known, headers starting
//   after the genesis block will be returned
//
// This function is safe for concurrent access.
//func locateHeaders(locator BlockLocator, hashStop *common.Hash) []wire.BlockHeader {
//	//b.chainLock.RLock()
//	headers := b.locateHeaders(locator, hashStop, wire.MaxBlockHeadersPerMsg)
//	//b.chainLock.RUnlock()
//	return headers
//}

// locateHeaders returns the headers of the blocks after the first known block
// in the locator until the provided stop hash is reached, or up to the provided
// max number of block headers.
//
// See the comment on the exported function for more details on special cases.
//
// This function MUST be called with the chain state lock held (for reads).
func (bk *blockKeeper) locateHeaders(locator []common.Hash, hashStop *common.Hash, maxHeaders uint32) []types.BlockHeader {
	// Find the node after the first known block in the locator and the
	// total number of nodes after it needed while respecting the stop hash
	// and max entries.
	node, total := bk.locateInventory(locator, hashStop, maxHeaders)
	if total == 0 {
		return nil
	}
	log.Info("start block:", node.Height, " total:", total)
	// Populate and return the found headers.
	headers := make([]types.BlockHeader, 0, total)
	for i := uint32(0); i < total; i++ {
		headers = append(headers, node.BlockHeader)
		node, _ = bk.chain.GetBlockByHeight(node.Height + 1) //Next(node)
	}
	return headers
}

// locateInventory returns the node of the block after the first known block in
// the locator along with the number of subsequent nodes needed to either reach
// the provided stop hash or the provided max number of entries.
//
// In addition, there are two special cases:
//
// - When no locators are provided, the stop hash is treated as a request for
//   that block, so it will either return the node associated with the stop hash
//   if it is known, or nil if it is unknown
// - When locators are provided, but none of them are known, nodes starting
//   after the genesis block will be returned
//
// This is primarily a helper function for the locateBlocks and locateHeaders
// functions.
//
// This function MUST be called with the chain state lock held (for reads).
func (bk *blockKeeper) locateInventory(locator []common.Hash, hashStop *common.Hash, maxEntries uint32) (*types.Block, uint32) {
	// There are no block locators so a specific block is being requested
	// as identified by the stop hash.
	//stopNode := b.index.LookupNode(hashStop)
	b32 := [32]byte{}
	copy(b32[:], hashStop.Bytes())
	hash := bc.NewHash(b32)
	stopNode, err := bk.chain.GetBlockByHash(&hash)
	if err != nil {
		log.Info("Can not find stop node!")
	}
	if len(locator) == 0 {
		if stopNode == nil {
			// No blocks with the stop hash were found so there is
			// nothing to do.
			return nil, 0
		}
		return stopNode, 1
	}

	// Find the most recent locator block hash in the main chain.  In the
	// case none of the hashes in the locator are in the main chain, fall
	// back to the genesis block.
	startNode, err := bk.chain.GetBlockByHeight(0)
	for _, hash := range locator {
		b32 := [32]byte{}
		copy(b32[:], hash.Bytes())
		bchash := bc.NewHash(b32)

		node, _ := bk.chain.GetBlockByHash(&bchash)
		if node != nil && bk.chain.InMainChain(node.Hash()) { //.Contains(node) {
			startNode = node
			break
		}
	}

	// Start at the block after the most recently known block.  When there
	// is no next block it means the most recently known block is the tip of
	// the best chain, so there is nothing more to do.
	startNode, err = bk.chain.GetBlockByHeight(startNode.Height + 1) //Next(startNode)
	if startNode == nil || err != nil {
		log.Info("locateInventory err:", err)
		return nil, 0
	}

	// Calculate how many entries are needed.
	total := uint32((bk.chain.BestBlockHeight() - startNode.Height) + 1)
	if stopNode != nil && bk.chain.InMainChain(stopNode.Hash()) &&
		stopNode.Height >= startNode.Height {
		total = uint32((stopNode.Height - startNode.Height) + 1)
	}
	if total > maxEntries {
		total = maxEntries
	}

	return startNode, total
}

func (bk *blockKeeper) getHeaders(peerID string, locator []*common.Hash, stopHash *common.Hash) error {
	bkPeer, ok := bk.peers.Peer(peerID)
	if !ok {
		log.Info("peer is not registered")
		return errPeerNotRegister
	}
	return bkPeer.PushGetHeadersMsg(locator, stopHash)
}

// handleHeadersMsg handles block header messages from all peers.  Headers are
// requested when performing a headers-first sync.
func (bk *blockKeeper) handleHeadersMsg(peerID string, headers []types.BlockHeader) (error, bool) {
	peer, exists := bk.peers.Peer(peerID) //sm.Peers().Peer(peer.id)
	if !exists {
		log.Warnf("Received headers message from unknown peer %s", peer)
		return errPeerDropped, false
	}

	// The remote peer is misbehaving if we didn't request headers.
	numHeaders := len(headers)
	// Nothing to do for an empty headers message.
	if numHeaders == 0 {
		return errEmptyHeaders, false
	}

	// Process all of the received headers ensuring each one connects to the
	// previous and that checkpoints match.
	receivedCheckpoint := false
	//var finalHash *bc.Hash
	for _, header := range headers {
		// Ensure there is a previous header to compare against.
		prevBlockHeader := bk.headerList.Back()
		if prevBlockHeader == nil {
			log.Warnf("Header list does not contain a previous" +
				"element as expected -- disconnecting peer")
			peer.swPeer.CloseConn()
			return errPeerMisbehave, receivedCheckpoint
		}

		// Ensure the header properly connects to the previous one and
		// add it to the list of headers.
		blockHash := header.Hash()
		node := headerNode{hash: common.CoreHashToHash(&blockHash)}
		prevNode := prevBlockHeader.Value.(*headerNode)
		if prevNode.hash.Str() == header.PreviousBlockHash.String() {
			node.height = prevNode.height + 1
			e := bk.headerList.PushBack(&node)
			if bk.startHeader == nil {
				bk.startHeader = e
			}
		} else {
			log.Warnf("Received block header that does not "+
				"properly connect to the chain from peer %s "+
				"-- disconnecting", peer.id)
			peer.swPeer.CloseConn()
			return errPeerMisbehave, receivedCheckpoint
		}

		// Verify the header at the next checkpoint height matches.
		if node.height == bk.nextCheckpoint().Height {
			if node.hash.Str() == bk.nextCheckpoint().Hash.String() {
				receivedCheckpoint = true
				log.Infof("Verified downloaded block "+
					"header against checkpoint at height "+
					"%d/hash %s", node.height, node.hash)
			} else {
				log.Warnf("Block header at height %d/hash "+
					"%s from peer %s does NOT match "+
					"expected checkpoint hash of %s -- "+
					"disconnecting", node.height,
					node.hash, peer.id,
					bk.nextCheckpoint().Hash.String())
				peer.swPeer.CloseConn()
				return errPeerMisbehave, receivedCheckpoint
			}
			break
		}
	}

	// When this header is a checkpoint, switch to fetching the blocks for
	// all of the headers since the last checkpoint.
	if receivedCheckpoint {
		// Since the first entry of the list is always the final block
		// that is already in the database and is only used to ensure
		// the next header links properly, it must be removed before
		// fetching the blocks.
		bk.headerList.Remove(bk.headerList.Front())
		log.Infof("Received %v block headers: Fetching blocks",
			bk.headerList.Len())
	}

	return nil, receivedCheckpoint
}

//// fetchHeaderBlocks creates and sends a request to the syncPeer for the next
//// list of blocks to be downloaded based on the current list of headers.
//func (sm *SyncManager) fetchHeaderBlocks() {
//	// Nothing to do if there is no start header.
//	if sm.startHeader == nil {
//		log.Warnf("fetchHeaderBlocks called with no start header")
//		return
//	}
//
//	// Build up a getdata request for the list of blocks the headers
//	// describe.  The size hint will be limited to wire.MaxInvPerMsg by
//	// the function, so no need to double check it here.
//	gdmsg := wire.NewMsgGetDataSizeHint(uint(sm.headerList.Len()))
//	numRequested := 0
//	for e := sm.startHeader; e != nil; e = e.Next() {
//		node, ok := e.Value.(*headerNode)
//		if !ok {
//			log.Warn("Header list node type is not a headerNode")
//			continue
//		}
//
//		iv := wire.NewInvVect(wire.InvTypeBlock, node.hash)
//		haveInv, err := sm.haveInventory(iv)
//		if err != nil {
//			log.Warnf("Unexpected failure when checking for "+
//				"existing inventory during header block "+
//				"fetch: %v", err)
//		}
//		if !haveInv {
//			syncPeerState := sm.peerStates[sm.syncPeer]
//
//			sm.requestedBlocks[*node.hash] = struct{}{}
//			syncPeerState.requestedBlocks[*node.hash] = struct{}{}
//
//			// If we're fetching from a witness enabled peer
//			// post-fork, then ensure that we receive all the
//			// witness data in the blocks.
//			if sm.syncPeer.IsWitnessEnabled() {
//				iv.Type = wire.InvTypeWitnessBlock
//			}
//
//			gdmsg.AddInvVect(iv)
//			numRequested++
//		}
//		sm.startHeader = e.Next()
//		if numRequested >= wire.MaxInvPerMsg {
//			break
//		}
//	}
//	if len(gdmsg.InvList) > 0 {
//		sm.syncPeer.QueueMessage(gdmsg, nil)
//	}
//}

// headersMsg packages a bitcoin headers message and the peer it came from
// together so the block handler has access to that information.
type headersMsg struct {
	headers []types.BlockHeader
	peerID  string
}

// headerNode is used as a node in a list of headers that are linked together
// between checkpoints.
type headerNode struct {
	height uint64
	hash   *common.Hash
}

// log2FloorMasks defines the masks to use when quickly calculating
// floor(log2(x)) in a constant log2(32) = 5 steps, where x is a uint32, using
// shifts.  They are derived from (2^(2^x) - 1) * (2^(2^x)), for x in 4..0.
var log2FloorMasks = []uint32{0xffff0000, 0xff00, 0xf0, 0xc, 0x2}

// fastLog2Floor calculates and returns floor(log2(x)) in a constant 5 steps.
func fastLog2Floor(n uint32) uint8 {
	rv := uint8(0)
	exponent := uint8(16)
	for i := 0; i < 5; i++ {
		if n&log2FloorMasks[i] != 0 {
			rv += exponent
			n >>= exponent
		}
		exponent >>= 1
	}
	return rv
}

// blockLocator returns a block locator for the passed block node.  The passed
// node can be nil in which case the block locator for the current tip
// associated with the view will be returned.  This only differs from the
// exported version in that it is up to the caller to ensure the lock is held.
//
// See the exported BlockLocator function comments for more details.
//
// This function MUST be called with the view mutex locked (for reads).
func (bk *blockKeeper) blockLocator(node *types.BlockHeader) []*common.Hash {
	// Use the current tip if requested.
	if node == nil {
		node = bk.chain.BestBlockHeader()
	}
	if node == nil {
		return nil
	}

	// Calculate the max number of entries that will ultimately be in the
	// block locator.  See the description of the algorithm for how these
	// numbers are derived.
	var maxEntries uint8
	if node.Height <= 12 {
		maxEntries = uint8(node.Height) + 1
	} else {
		// Requested hash itself + previous 10 entries + genesis block.
		// Then floor(log2(height-10)) entries for the skip portion.
		adjustedHeight := uint32(node.Height) - 10
		maxEntries = 12 + fastLog2Floor(adjustedHeight)
	}
	locator := make([]*common.Hash, 0, maxEntries)

	step := uint64(1)
	for node != nil {
		hash := common.Hash(node.Hash().Byte32())
		fmt.Println("height:", node.Height, " hash:", hash)

		locator = append(locator, &hash)

		// Nothing more to add once the genesis block has been added.
		if node.Height == 0 {
			break
		}

		// Calculate height of previous node to include ensuring the
		// final node is the genesis block.
		height := node.Height - step
		if node.Height < step {
			height = 0
		}

		// When the node is in the current chain view, all of its
		// ancestors must be too, so use a much faster O(1) lookup in
		// that case.  Otherwise, fall back to walking backwards through
		// the nodes of the other chain to the correct ancestor.
		block, _ := bk.chain.GetBlockByHeight(height)
		node = &block.BlockHeader
		//if c.contains(node) {
		//	node = c.nodes[height]
		//} else {
		//	node = node.Ancestor(height)
		//}

		// Once 11 entries have been included, start doubling the
		// distance between included hashes.
		if len(locator) > 10 {
			step *= 2
		}
	}

	return locator
}

// handleHeadersMsg handles block header messages from all peers.  Headers are
// requested when performing a headers-first sync.
func (bk *blockKeeper) GetBlocksWorker(peerID string, bmsg *GetBlocksMessage) {
	peer, exists := bk.peers.Peer(peerID) //sm.Peers().Peer(peer.id)
	if !exists {
		log.Warnf("Received blocks message from unknown peer %s", peer)
		return
	}
	var beginBlock *types.Block
	beginHash := bc.NewHash(bmsg.beginHash)
	if beginBlock, _ = bk.chain.GetBlockByHash(&beginHash); beginBlock != nil {
		log.Error("GetBlocks Worker can't find begin Hash")
		return
	}

	msg, _ := NewBlocksMessage()
	for height := beginBlock.Height; height < beginBlock.Height+uint64(bmsg.num); height++ {
		block, _ := bk.chain.GetBlockByHeight(height)
		hash := block.Hash().Byte32()
		rawBlock, _ := block.MarshalText()
		blockMsg := blockMsg{Hash: hash, RawBlock: rawBlock}
		msg.blocks = append(msg.blocks, blockMsg)
	}
	bk.blocksSend(peerID, *msg)
}

func (bk *blockKeeper) blocksSend(peerID string, msg BlocksMessage) error {
	return bk.peers.SendBlocks(peerID, msg)
}
