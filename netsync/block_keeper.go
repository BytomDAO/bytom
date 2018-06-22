package netsync

import (
	"container/list"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/common"
	"github.com/bytom/consensus"
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

	maxTxChanSize = 10000 // txChanSize is the size of channel listening to Txpool newTxCh
)

var (
	errGetBlockTimeout = errors.New("Get block Timeout")
	errPeerDropped     = errors.New("Peer dropped")
	errGetBlockByHash  = errors.New("Get block by hash error")
	errBroadcastStatus = errors.New("Broadcast new status block error")
	errReqBlock        = errors.New("Request block error")
	errPeerNotRegister = errors.New("peer is not registered")
)

//TODO: add retry mechanism
type blockKeeper struct {
	chain *protocol.Chain
	sw    *p2p.Switch
	peers *peerSet

	pendingProcessCh chan *blockPending
	txsProcessCh     chan *txsNotify
	quitReqBlockCh   chan *string
	headersFirstMode bool
	headerList       *list.List
	startHeader      *list.Element
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

func (bk *blockKeeper) headersSend(peerID string, headers []*types.BlockHeader) error {
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
	headers := bk.locateHeaders(msg.BlockLocatorHashes, &msg.HashStop, MaxBlockHeadersPerMsg)
	if len(headers) == 0 {
		// Nothing to send.
		return
	}

	// Send found headers to the requesting peer.
	blockHeaders := make([]*types.BlockHeader, len(headers))
	for i := range headers {
		blockHeaders[i] = &headers[i]
	}
	bk.headersSend(peerID, blockHeaders)
	//sp.QueueMessage(&HeadersMessage{Headers: blockHeaders}, nil)
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
func (bk *blockKeeper) locateHeaders(locator []*common.Hash, hashStop *common.Hash, maxHeaders uint32) []types.BlockHeader {
	// Find the node after the first known block in the locator and the
	// total number of nodes after it needed while respecting the stop hash
	// and max entries.
	node, total := bk.locateInventory(locator, hashStop, maxHeaders)
	if total == 0 {
		return nil
	}

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
func (bk *blockKeeper) locateInventory(locator []*common.Hash, hashStop *common.Hash, maxEntries uint32) (*types.Block, uint32) {
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
func (bk *blockKeeper) handleHeadersMsg(hmsg *headersMsg) {
	peer := hmsg.peer
	_, exists := bk.peers.Peer(peer.id) //sm.Peers().Peer(peer.id)
	if !exists {
		log.Warnf("Received headers message from unknown peer %s", peer)
		return
	}

	// The remote peer is misbehaving if we didn't request headers.
	//msg := hmsg.headers
	numHeaders := len(hmsg.headers)
	if !bk.headersFirstMode {
		log.Warnf("Got %d unrequested headers from %s -- "+
			"disconnecting", numHeaders, peer.id)
		peer.swPeer.CloseConn()
		return
	}

	// Nothing to do for an empty headers message.
	if numHeaders == 0 {
		return
	}

	// Process all of the received headers ensuring each one connects to the
	// previous and that checkpoints match.
	receivedCheckpoint := false
	var finalHash *bc.Hash
	for _, blockHeader := range hmsg.headers {
		blockHash := blockHeader.Hash()
		finalHash = &blockHash

		// Ensure there is a previous header to compare against.
		prevNodeEl := bk.headerList.Back()
		if prevNodeEl == nil {
			log.Warnf("Header list does not contain a previous" +
				"element as expected -- disconnecting peer")
			peer.swPeer.CloseConn()
			return
		}

		// Ensure the header properly connects to the previous one and
		// add it to the list of headers.
		node := headerNode{hash: &blockHash}
		prevNode := prevNodeEl.Value.(*headerNode)
		if prevNode.hash.String() == blockHeader.PreviousBlockHash.String() {
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
			return
		}

		// Verify the header at the next checkpoint height matches.
		if node.height == bk.nextCheckpoint().Height {
			if node.hash.String() == bk.nextCheckpoint().Hash.String() {
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
				return
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
		//todo:
		//sm.fetchHeaderBlocks()
		return
	}

	// This header is not a checkpoint, so request the next batch of
	// headers starting from the latest known header and ending with the
	// next checkpoint.
	hash := common.Hash(finalHash.Byte32())
	locator := []*common.Hash{&hash}
	stopHash := common.BytesToHash(bk.nextCheckpoint().Hash.Bytes())
	err := peer.PushGetHeadersMsg(locator, &stopHash)
	if err != nil {
		log.Warnf("Failed to send getheaders message to "+"peer %s: %v", peer.id, err)
		return
	}
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
	headers []*types.BlockHeader
	peer    *peer
}

// headerNode is used as a node in a list of headers that are linked together
// between checkpoints.
type headerNode struct {
	height uint64
	hash   *bc.Hash
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
		fmt.Println("height:", node.Height)
		hash := common.Hash(node.Hash().Byte32())
		locator = append(locator, &hash)

		// Nothing more to add once the genesis block has been added.
		if node.Height == 0 {
			break
		}

		// Calculate height of previous node to include ensuring the
		// final node is the genesis block.
		height := node.Height - step
		if height < 0 {
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
