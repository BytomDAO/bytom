package consensusmgr

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"

	"github.com/bytom/bytom/p2p/security"
	"github.com/bytom/bytom/protocol/bc"
)

const (
	maxBlockDistance = 64
	newBlockChSize   = 64
	msgLimit         = 128 // peer message number limit
)

// blockFetcher is responsible for accumulating block announcements from various peers
// and scheduling them for retrieval.
type blockFetcher struct {
	chain Chain
	peers Peers

	newBlockCh chan *blockMsg
	queue      *prque.Prque          // block import priority queue
	msgSet     map[bc.Hash]*blockMsg // already queued blocks
	msgCounter map[string]int        // per peer msg counter to prevent DOS
}

//NewBlockFetcher creates a block fetcher to retrieve blocks of the new propose.
func newBlockFetcher(chain Chain, peers Peers) *blockFetcher {
	return &blockFetcher{
		chain:      chain,
		peers:      peers,
		newBlockCh: make(chan *blockMsg, newBlockChSize),
		queue:      prque.New(),
		msgSet:     make(map[bc.Hash]*blockMsg),
		msgCounter: make(map[string]int),
	}
}

func (f *blockFetcher) blockProcessorLoop() {
	for {
		for !f.queue.Empty() {
			msg := f.queue.PopItem().(*blockMsg)
			if msg.block.Height > f.chain.BestBlockHeight()+1 {
				f.queue.Push(msg, -float32(msg.block.Height))
				break
			}

			f.insert(msg)
			delete(f.msgSet, msg.block.Hash())
			f.msgCounter[msg.peerID]--
			if f.msgCounter[msg.peerID] <= 0 {
				delete(f.msgCounter, msg.peerID)
			}
		}
		f.add(<-f.newBlockCh, msgLimit)
	}
}

func (f *blockFetcher) add(msg *blockMsg, limit int) {
	// prevent DOS
	count := f.msgCounter[msg.peerID] + 1
	if count > limit {
		log.WithFields(log.Fields{"module": logModule, "peer": msg.peerID, "limit": limit}).Warn("The number of peer messages exceeds the limit")
		return
	}

	bestHeight := f.chain.BestBlockHeight()
	if bestHeight > msg.block.Height || msg.block.Height-bestHeight > maxBlockDistance {
		return
	}

	blockHash := msg.block.Hash()
	if _, ok := f.msgSet[blockHash]; !ok {
		f.msgSet[blockHash] = msg
		f.queue.Push(msg, -float32(msg.block.Height))
		f.msgCounter[msg.peerID] = count
		log.WithFields(log.Fields{
			"module":       logModule,
			"block height": msg.block.Height,
			"block hash":   blockHash.String(),
		}).Debug("blockFetcher receive propose block")
	}
}

func (f *blockFetcher) insert(msg *blockMsg) {
	isOrphan, err := f.chain.ProcessBlock(msg.block)
	if err != nil {
		peer := f.peers.GetPeer(msg.peerID)
		if peer == nil {
			return
		}
		f.peers.ProcessIllegal(msg.peerID, security.LevelMsgIllegal, err.Error())
		return
	}

	if isOrphan {
		return
	}

	proposeMsg, err := NewBlockProposeMsg(msg.block)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on create BlockProposeMsg")
		return
	}

	if err := f.peers.BroadcastMsg(NewBroadcastMsg(proposeMsg, consensusChannel)); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on broadcast proposed block")
		return
	}
}

func (f *blockFetcher) processNewBlock(msg *blockMsg) {
	f.newBlockCh <- msg
}
