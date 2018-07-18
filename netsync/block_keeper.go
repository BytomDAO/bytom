package netsync

import (
	"container/list"
	"time"

	"github.com/bytom/consensus"
	"github.com/bytom/errors"
	"github.com/bytom/mining/tensority"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	syncTimeout        = 30 * time.Second
	requestRetryTicker = 15 * time.Second

	maxBlocksPending      = 1024
	maxtxsPending         = 32768
	maxHeadersPending     = 32
	maxBlockHeadersPerMsg = 2000

	maxQuitReq = 256

	maxTxChanSize          = 10000 // txChanSize is the size of channel listening to Txpool newTxCh
	maxRequestBlocksPerMsg = 20
	MaxMsgPackageSize      = 10 * 1024 * 1024
)

var (
	errRequestTimeout   = errors.New("request timeout")
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

//TODO: add retry mechanism
type blockKeeper struct {
	chain    *protocol.Chain
	peers    *peerSet
	syncPeer *peer

	blockProcessCh   chan *blockMsg
	blocksProcessCh  chan *blocksMsg
	headersProcessCh chan *headersMsg

	txsProcessCh   chan *txMsg
	quitReqBlockCh chan *string
	headerList     *list.List
	startHeader    *list.Element
}

func newBlockKeeper(chain *protocol.Chain, peers *peerSet, quitReqBlockCh chan *string) *blockKeeper {
	best := chain.BestBlockHeader()
	bk := &blockKeeper{
		chain:            chain,
		peers:            peers,
		blockProcessCh:   make(chan *blockMsg, maxBlocksPending),
		txsProcessCh:     make(chan *txMsg, maxtxsPending),
		headersProcessCh: make(chan *headersMsg, maxHeadersPending),
		blocksProcessCh:  make(chan *blocksMsg, 0),
		quitReqBlockCh:   quitReqBlockCh,
		headerList:       list.New(),
	}
	bk.resetHeaderState(best)
	go bk.txsProcessWorker()
	return bk
}

// resetHeaderState sets the headers-first mode state to values appropriate for
// syncing from a new peer.
func (bk *blockKeeper) resetHeaderState(header *types.BlockHeader) {
	bk.headerList.Init()
	bk.startHeader = nil

	// When there is a next checkpoint, add an entry for the latest known
	// block into the header pool.  This allows the next downloaded header
	// to prove it links to the chain properly.
	if bk.nextCheckpoint() != nil {
		bk.headerList.PushBack(header)
	}
}

func (bk *blockKeeper) AddBlock(block *types.Block, peerID string) {
	bk.blockProcessCh <- &blockMsg{block: block, peerID: peerID}
}

func (bk *blockKeeper) AddTx(tx *types.Tx, peerID string) {
	bk.txsProcessCh <- &txMsg{tx: tx, peerID: peerID}
}

func (bk *blockKeeper) IsCaughtUp() bool {
	peer := bk.peers.BestPeer(consensus.SFFullNode)
	return peer == nil || bk.chain.BestBlockHeight() >= peer.Height()
}

func (bk *blockKeeper) appendHeaderList(headers []*types.BlockHeader) error {
	for _, header := range headers {
		prevHeader := bk.headerList.Back().Value.(*types.BlockHeader)
		if prevHeader.Hash() != header.PreviousBlockHash {
			return errPeerMisbehave
		}

		bk.headerList.PushBack(header)
	}
	return nil
}

func (bk *blockKeeper) blockLocator() []*bc.Hash {
	header := bk.chain.BestBlockHeader()
	locator := []*bc.Hash{}

	step := uint64(1)
	for header != nil {
		headerHash := header.Hash()
		locator = append(locator, &headerHash)
		if header.Height == 0 {
			break
		}

		if header.Height < step {
			header, _ = bk.chain.GetHeaderByHeight(0)
		} else {
			header, _ = bk.chain.GetHeaderByHeight(header.Height - step)
		}

		if len(locator) > 10 {
			step *= 2
		}
	}
	return locator
}

func (bk *blockKeeper) fastBlockSync(checkPoint *consensus.Checkpoint) error {
	for bk.headerList.Back().Value.(*types.BlockHeader).Hash() != checkPoint.Hash {
		locator := bk.blockLocator()
		headers, err := bk.requireHeaders(locator)
		if err != nil {
			return err
		}

		if err := bk.appendHeaderList(headers); err != nil {
			return err
		}
	}

	for bk.chain.BestBlockHeight() < checkPoint.Height {
		locator := bk.blockLocator()
		blocks, err := bk.requireBlocks(locator, &checkPoint.Hash)
		if err != nil {
			return err
		}

		for _, block := range blocks {
			blockHash := block.Hash()
			if blockHash != bk.startHeader.Value.(*types.BlockHeader).Hash() {
				return errPeerMisbehave
			}

			seed, err := bk.chain.CalcNextSeed(&(block.PreviousBlockHash))
			if err != nil {
				return err
			}

			tensority.AIHash.AddCache(&blockHash, seed, &bc.Hash{})
			isOrphan, err := bk.chain.ProcessBlock(block)
			tensority.AIHash.RemoveCache(&blockHash, seed)
			if err != nil || isOrphan {
				return errPeerMisbehave
			}
		}
	}
	return nil
}

func (bk *blockKeeper) requireBlock(height uint64) (*types.Block, error) {
	if ok := bk.syncPeer.getBlockByHeight(height); !ok {
		return nil, errPeerDropped
	}

	waitTicker := time.NewTimer(syncTimeout)
	for {
		select {
		case msg := <-bk.blockProcessCh:
			if msg.block.Height != height {
				bk.syncPeer.addBanScore(0, 2, "require block got different height")
				continue
			}
			return msg.block, nil
		case <-waitTicker.C:
			return nil, errRequestTimeout
		}
	}
}

func (bk *blockKeeper) requireBlocks(locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error) {
	if ok := bk.syncPeer.getBlocks(locator, stopHash); !ok {
		return nil, errPeerDropped
	}

	waitTicker := time.NewTimer(syncTimeout)
	for {
		select {
		case msg := <-bk.blocksProcessCh:
			return msg.blocks, nil
		case <-waitTicker.C:
			return nil, errGetBlocksTimeout
		}
	}
}

func (bk *blockKeeper) requireHeaders(locator []*bc.Hash) ([]*types.BlockHeader, error) {
	stopHash := bk.nextCheckpoint().Hash
	bk.syncPeer.getHeaders(locator, &stopHash)

	waitTicker := time.NewTimer(syncTimeout)
	for {
		select {
		case msg := <-bk.headersProcessCh:
			return msg.headers, nil
		case <-waitTicker.C:
			return nil, errRequestTimeout
		}
	}
}

func (bk *blockKeeper) regularBlockSync(wantHeight uint64) error {
	i := bk.chain.BestBlockHeight() + 1
	for i <= wantHeight {
		block, err := bk.requireBlock(i)
		if err != nil {
			return err
		}

		isOrphan, err := bk.chain.ProcessBlock(block)
		if err != nil {
			return err
		}

		if isOrphan {
			i--
			continue
		}
		i = bk.chain.BestBlockHeight() + 1
	}
	return nil
}

func (bk *blockKeeper) startSync() {
	checkPoint := bk.nextCheckpoint()
	peer := bk.peers.BestPeer(consensus.SFFastSync | consensus.SFFullNode)
	if peer != nil && checkPoint != nil && peer.Height() >= checkPoint.Height {
		bk.syncPeer = peer
		if err := bk.fastBlockSync(checkPoint); err != nil {
			bk.peers.StopPeerGracefully(peer.ID())
		}
		return
	}

	peer = bk.peers.BestPeer(consensus.SFFullNode)
	if peer.Height() > bk.chain.BestBlockHeight() {
		bk.syncPeer = peer
		if err := bk.regularBlockSync(peer.Height()); err != nil {
			bk.peers.StopPeerGracefully(peer.ID())
		}
		return
	}
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

func (bk *blockKeeper) txsProcessWorker() {
	for txsResponse := range bk.txsProcessCh {
		tx := txsResponse.tx

		bkPeer := bk.peers.getPeer(txsResponse.peerID)
		if bkPeer == nil {
			continue
		}

		bkPeer.markTransaction(&tx.ID)
		if isOrphan, err := bk.chain.ValidateTx(tx); err != nil && isOrphan == false {
			if ban := bkPeer.addBanScore(10, 0, "tx error"); ban {
				bk.peers.AddBannedPeer(txsResponse.peerID)
				bk.peers.StopPeerGracefully(txsResponse.peerID)
			}
		}
	}
}

func (bk *blockKeeper) locateBlocks(locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error) {
	headers, err := bk.locateHeaders(locator, stopHash)
	if err != nil {
		return nil, err
	}

	blocks := []*types.Block{}
	for _, headers := range headers {
		block, err := bk.chain.GetBlockByHeight(headers.Height)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (bk *blockKeeper) locateHeaders(locator []*bc.Hash, stopHash *bc.Hash) ([]*types.BlockHeader, error) {
	stopHeader, err := bk.chain.GetHeaderByHash(stopHash)
	if err != nil {
		return nil, err
	}

	startHeader, err := bk.chain.GetHeaderByHeight(0)
	if err != nil {
		return nil, err
	}

	for _, hash := range locator {
		header, err := bk.chain.GetHeaderByHash(hash)
		if err != nil && bk.chain.InMainChain(header.Hash()) {
			startHeader = header
			break
		}
	}

	totalHeaders := stopHeader.Height - startHeader.Height
	if totalHeaders > maxBlockHeadersPerMsg {
		totalHeaders = maxBlockHeadersPerMsg
	}

	headers := []*types.BlockHeader{}
	for i := uint64(0); i < totalHeaders; i++ {
		header, err := bk.chain.GetHeaderByHeight(startHeader.Height + i)
		if err != nil {
			return nil, err
		}

		headers = append(headers, header)
	}
	return headers, nil
}
