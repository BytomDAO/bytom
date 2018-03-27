package netsync

import (
	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"
	"reflect"

	"fmt"
	"github.com/bytom/blockchain/account"
	"github.com/bytom/mining/cpuminer"
	"github.com/bytom/mining/miningpool"
	"github.com/bytom/netsync/fetcher"
	"github.com/bytom/p2p"
	"github.com/bytom/p2p/trust"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/types"
)

const (
	// BlockchainChannel is a channel for blocks and status updates
	BlockchainChannel = byte(0x40)
	maxNewBlockChSize = int(1024)

	maxBlockchainResponseSize = 22020096 + 2
)

const (
	// SUCCESS indicates the rpc calling is successful.
	SUCCESS = "success"
	// FAIL indicated the rpc calling is failed.
	FAIL = "fail"
)

// Response describes the response standard.
type Response struct {
	Status string      `json:"status,omitempty"`
	Msg    string      `json:"msg,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

//NewSuccessResponse success response
func NewSuccessResponse(data interface{}) Response {
	return Response{Status: SUCCESS, Data: data}
}

//NewErrorResponse error response
func NewErrorResponse(err error) Response {
	return Response{Status: FAIL, Msg: err.Error()}
}

//ProtocalReactor handles long-term catchup syncing.
type ProtocalReactor struct {
	p2p.BaseReactor

	chain        *protocol.Chain
	blockKeeper  *blockKeeper
	txPool       *protocol.TxPool
	mining       *cpuminer.CPUMiner
	miningPool   *miningpool.MiningPool
	sw           *p2p.Switch
	evsw         types.EventSwitch
	miningEnable bool
	fetcher      *fetcher.Fetcher

	newPeerCh chan struct{}
}

// NewProtocalReactor returns the reactor of whole blockchain.
func NewProtocalReactor(chain *protocol.Chain, txPool *protocol.TxPool, accounts *account.Manager, sw *p2p.Switch, miningEnable bool, newBlockCh chan *bc.Hash, fetcher *fetcher.Fetcher) *ProtocalReactor {
	pr := &ProtocalReactor{
		chain:        chain,
		blockKeeper:  newBlockKeeper(chain, sw),
		txPool:       txPool,
		mining:       cpuminer.NewCPUMiner(chain, accounts, txPool, newBlockCh),
		miningPool:   miningpool.NewMiningPool(chain, accounts, txPool, newBlockCh),
		sw:           sw,
		miningEnable: miningEnable,
		fetcher:      fetcher,
		newPeerCh:    make(chan struct{}),
	}
	pr.BaseReactor = *p2p.NewBaseReactor("ProtocalReactor", pr)
	return pr
}

// GetChannels implements Reactor
func (pr *ProtocalReactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		&p2p.ChannelDescriptor{
			ID:                BlockchainChannel,
			Priority:          5,
			SendQueueCapacity: 100,
		},
	}
}

// GetChannels implements Reactor
func (pr *ProtocalReactor) GetNewPeerChan() *chan struct{} {
	return &pr.newPeerCh
}

// OnStart implements BaseService
func (pr *ProtocalReactor) OnStart() error {
	pr.BaseReactor.OnStart()

	if pr.miningEnable {
		pr.mining.Start()
	}
	return nil
}

// OnStop implements BaseService
func (pr *ProtocalReactor) OnStop() {
	pr.BaseReactor.OnStop()
	if pr.miningEnable {
		pr.mining.Stop()
	}
	pr.blockKeeper.Stop()
}

// AddPeer implements Reactor by sending our state to peer.
func (pr *ProtocalReactor) AddPeer(peer *p2p.Peer) {
	pr.blockKeeper.AddPeer(peer)
	peer.Send(BlockchainChannel, struct{ BlockchainMessage }{&StatusRequestMessage{}})
}

// RemovePeer implements Reactor by removing peer from the pool.
func (pr *ProtocalReactor) RemovePeer(peer *p2p.Peer, reason interface{}) {
	pr.blockKeeper.RemovePeer(peer.Key)
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (pr *ProtocalReactor) Receive(chID byte, src *p2p.Peer, msgBytes []byte) {
	var tm *trust.TrustMetric
	key := src.Connection().RemoteAddress.IP.String()
	if tm = pr.sw.TrustMetricStore.GetPeerTrustMetric(key); tm == nil {
		log.Errorf("Can't get peer trust metric")
		return
	}

	_, msg, err := DecodeMessage(msgBytes)
	if err != nil {
		log.Errorf("Error decoding messagek %v", err)
		return
	}
	log.WithFields(log.Fields{"peerID": src.Key, "msg": msg}).Info("Receive request")

	switch msg := msg.(type) {
	case *BlockRequestMessage:
		var block *legacy.Block
		var err error
		if msg.Height != 0 {
			block, err = pr.chain.GetBlockByHeight(msg.Height)
		} else {
			block, err = pr.chain.GetBlockByHash(msg.GetHash())
		}
		if err != nil {
			log.Errorf("Fail on BlockRequestMessage get block: %v", err)
			return
		}
		response, err := NewBlockResponseMessage(block)
		if err != nil {
			log.Errorf("Fail on BlockRequestMessage create resoinse: %v", err)
			return
		}
		src.TrySend(BlockchainChannel, struct{ BlockchainMessage }{response})

	case *BlockResponseMessage:
		fmt.Println("BlockResponseMessage height:", msg.GetBlock().Height)
		pr.blockKeeper.AddBlock(msg.GetBlock(), src)

	case *StatusRequestMessage:
		block := pr.chain.BestBlock()
		src.TrySend(BlockchainChannel, struct{ BlockchainMessage }{NewStatusResponseMessage(block)})

	case *StatusResponseMessage:
		pr.blockKeeper.SetPeerHeight(src.Key, msg.Height, msg.GetHash())
		pr.newPeerCh <- struct{}{}

	case *TransactionNotifyMessage:
		//TODO: test
		tx, err := msg.GetTransaction()
		if err != nil {
			log.Errorf("Error decoding new tx %v", err)
			return
		}
		pr.blockKeeper.MarkTransaction(src.Key, tx.ID.Byte32())
		if err := pr.chain.ValidateTx(tx); err != nil {
			pr.sw.AddScamPeer(src)
		}

	case *MineBlockMessage:
		block, err := msg.GetMineBlock()
		if err != nil {
			log.Errorf("Error decoding mined block %v", err)
			return
		}
		// Mark the peer as owning the block and schedule it for import
		pr.blockKeeper.MarkBlock(src.Key, block.Hash().Byte32())
		pr.fetcher.Enqueue(src.Key, block)
		hash := block.Hash()
		pr.blockKeeper.peers[src.Key].SetStatus(block.Height, &hash)

	default:
		log.Error(cmn.Fmt("Unknown message type %v", reflect.TypeOf(msg)))
	}
}

// BroadcastStatusResponse broadcasts `BlockStore` height.
func (pr *ProtocalReactor) BroadcastStatusResponse() {
	block := pr.chain.BestBlock()
	pr.Switch.Broadcast(BlockchainChannel, struct{ BlockchainMessage }{NewStatusResponseMessage(block)})
}

// BroadcastTransaction broadcats `BlockStore` transaction.
func (pr *ProtocalReactor) BroadcastTransaction(tx *legacy.Tx) error {
	msg, err := NewTransactionNotifyMessage(tx)
	if err != nil {
		return err
	}
	pr.Switch.Broadcast(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}
