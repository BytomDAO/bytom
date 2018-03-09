package blockchain

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/bytom/blockchain/accesstoken"
	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txfeed"
	"github.com/bytom/blockchain/wallet"
	"github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/mining/cpuminer"
	"github.com/bytom/mining/miningpool"
	"github.com/bytom/p2p"
	"github.com/bytom/p2p/trust"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/types"
)

const (
	// BlockchainChannel is a channel for blocks and status updates
	BlockchainChannel = byte(0x40)

	defaultChannelCapacity      = 100
	trySyncIntervalMS           = 100
	statusUpdateIntervalSeconds = 10
	maxBlockchainResponseSize   = 22020096 + 2
	crosscoreRPCPrefix          = "/rpc/"
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

//BlockchainReactor handles long-term catchup syncing.
type BlockchainReactor struct {
	p2p.BaseReactor

	chain         *protocol.Chain
	wallet        *wallet.Wallet
	accounts      *account.Manager
	assets        *asset.Registry
	accessTokens  *accesstoken.CredentialStore
	txFeedTracker *txfeed.Tracker
	blockKeeper   *blockKeeper
	txPool        *protocol.TxPool
	hsm           *pseudohsm.HSM
	mining        *cpuminer.CPUMiner
	miningPool    *miningpool.MiningPool
	mux           *http.ServeMux
	sw            *p2p.Switch
	handler       http.Handler
	evsw          types.EventSwitch
	miningEnable  bool
}

func batchRecover(ctx context.Context, v *interface{}) {
	if r := recover(); r != nil {
		var err error
		if recoveredErr, ok := r.(error); ok {
			err = recoveredErr
		} else {
			err = fmt.Errorf("panic with %T", r)
		}
		err = errors.Wrap(err)
		*v = err
	}

	if *v == nil {
		return
	}
	// Convert errors into error responses (including errors
	// from recovered panics above).
	if err, ok := (*v).(error); ok {
		*v = errorFormatter.Format(err)
	}
}

func (bcr *BlockchainReactor) info(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"is_configured": false,
		"version":       "0.001",
		"build_commit":  "----",
		"build_date":    "------",
		"build_config":  "---------",
	}, nil
}

func maxBytes(h http.Handler) http.Handler {
	const maxReqSize = 1e7 // 10MB
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// A block can easily be bigger than maxReqSize, but everything
		// else should be pretty small.
		if req.URL.Path != crosscoreRPCPrefix+"signer/sign-block" {
			req.Body = http.MaxBytesReader(w, req.Body, maxReqSize)
		}
		h.ServeHTTP(w, req)
	})
}

// Used as a request object for api queries
type requestQuery struct {
	Filter       string        `json:"filter,omitempty"`
	FilterParams []interface{} `json:"filter_params,omitempty"`
	SumBy        []string      `json:"sum_by,omitempty"`
	PageSize     int           `json:"page_size"`

	// AscLongPoll and Timeout are used by /list-transactions
	// to facilitate notifications.
	AscLongPoll bool          `json:"ascending_with_long_poll,omitempty"`
	Timeout     json.Duration `json:"timeout"`

	// After is a completely opaque cursor, indicating that only
	// items in the result set after the one identified by `After`
	// should be included. It has no relationship to time.
	After string `json:"after"`

	// These two are used for time-range queries like /list-transactions
	StartTimeMS uint64 `json:"start_time,omitempty"`
	EndTimeMS   uint64 `json:"end_time,omitempty"`

	// This is used for point-in-time queries like /list-balances
	// TODO(bobg): Different request structs for endpoints with different needs
	TimestampMS uint64 `json:"timestamp,omitempty"`

	// This is used for filtering results from /list-access-tokens
	// Value must be "client" or "network"
	Type string `json:"type"`

	// Aliases is used to filter results from /mockshm/list-keys
	Aliases []string `json:"aliases,omitempty"`
}

// Used as a response object for api queries
type page struct {
	Items    interface{}  `json:"items"`
	Next     requestQuery `json:"next"`
	LastPage bool         `json:"last_page"`
	After    string       `json:"after"`
}

// NewBlockchainReactor returns the reactor of whole blockchain.
func NewBlockchainReactor(chain *protocol.Chain, txPool *protocol.TxPool, accounts *account.Manager, assets *asset.Registry, sw *p2p.Switch, hsm *pseudohsm.HSM, wallet *wallet.Wallet, txfeeds *txfeed.Tracker, accessTokens *accesstoken.CredentialStore, miningEnable bool) *BlockchainReactor {
	bcr := &BlockchainReactor{
		chain:         chain,
		wallet:        wallet,
		accounts:      accounts,
		assets:        assets,
		blockKeeper:   newBlockKeeper(chain, sw),
		txPool:        txPool,
		mining:        cpuminer.NewCPUMiner(chain, accounts, txPool),
		miningPool:    miningpool.NewMiningPool(chain, accounts, txPool),
		mux:           http.NewServeMux(),
		sw:            sw,
		hsm:           hsm,
		txFeedTracker: txfeeds,
		accessTokens:  accessTokens,
		miningEnable:  miningEnable,
	}
	bcr.BaseReactor = *p2p.NewBaseReactor("BlockchainReactor", bcr)
	return bcr
}

// OnStart implements BaseService
func (bcr *BlockchainReactor) OnStart() error {
	bcr.BaseReactor.OnStart()
	bcr.BuildHandler()

	if bcr.miningEnable {
		bcr.mining.Start()
	}
	go bcr.syncRoutine()
	return nil
}

// OnStop implements BaseService
func (bcr *BlockchainReactor) OnStop() {
	bcr.BaseReactor.OnStop()
	if bcr.miningEnable {
		bcr.mining.Stop()
	}
	bcr.blockKeeper.Stop()
}

// GetChannels implements Reactor
func (bcr *BlockchainReactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		&p2p.ChannelDescriptor{
			ID:                BlockchainChannel,
			Priority:          5,
			SendQueueCapacity: 100,
		},
	}
}

// AddPeer implements Reactor by sending our state to peer.
func (bcr *BlockchainReactor) AddPeer(peer *p2p.Peer) {
	peer.Send(BlockchainChannel, struct{ BlockchainMessage }{&StatusRequestMessage{}})
}

// RemovePeer implements Reactor by removing peer from the pool.
func (bcr *BlockchainReactor) RemovePeer(peer *p2p.Peer, reason interface{}) {
	bcr.blockKeeper.RemovePeer(peer.Key)
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (bcr *BlockchainReactor) Receive(chID byte, src *p2p.Peer, msgBytes []byte) {
	var tm *trust.TrustMetric
	key := src.Connection().RemoteAddress.IP.String()
	if tm = bcr.sw.TrustMetricStore.GetPeerTrustMetric(key); tm == nil {
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
			block, err = bcr.chain.GetBlockByHeight(msg.Height)
		} else {
			block, err = bcr.chain.GetBlockByHash(msg.GetHash())
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
		bcr.blockKeeper.AddBlock(msg.GetBlock(), src)

	case *StatusRequestMessage:
		block := bcr.chain.BestBlock()
		src.TrySend(BlockchainChannel, struct{ BlockchainMessage }{NewStatusResponseMessage(block)})

	case *StatusResponseMessage:
		bcr.blockKeeper.SetPeerHeight(src.Key, msg.Height, msg.GetHash())

	case *TransactionNotifyMessage:
		tx := msg.GetTransaction()
		if err := bcr.chain.ValidateTx(tx); err != nil {
			bcr.sw.AddScamPeer(src)
		}

	default:
		log.Error(cmn.Fmt("Unknown message type %v", reflect.TypeOf(msg)))
	}
}

// Handle messages from the poolReactor telling the reactor what to do.
// NOTE: Don't sleep in the FOR_LOOP or otherwise slow it down!
// (Except for the SYNC_LOOP, which is the primary purpose and must be synchronous.)
func (bcr *BlockchainReactor) syncRoutine() {
	statusUpdateTicker := time.NewTicker(statusUpdateIntervalSeconds * time.Second)
	newTxCh := bcr.txPool.GetNewTxCh()

	for {
		select {
		case newTx := <-newTxCh:
			bcr.txFeedTracker.TxFilter(newTx)
			go bcr.BroadcastTransaction(newTx)
		case _ = <-statusUpdateTicker.C:
			go bcr.BroadcastStatusResponse()

			if bcr.miningEnable {
				// mining if and only if block sync is finished
				if bcr.blockKeeper.IsCaughtUp() {
					bcr.mining.Start()
				} else {
					bcr.mining.Stop()
				}
			}
		case <-bcr.Quit:
			return
		}
	}
}

// BroadcastStatusResponse broadcasts `BlockStore` height.
func (bcr *BlockchainReactor) BroadcastStatusResponse() {
	block := bcr.chain.BestBlock()
	bcr.Switch.Broadcast(BlockchainChannel, struct{ BlockchainMessage }{NewStatusResponseMessage(block)})
}

// BroadcastTransaction broadcats `BlockStore` transaction.
func (bcr *BlockchainReactor) BroadcastTransaction(tx *legacy.Tx) error {
	msg, err := NewTransactionNotifyMessage(tx)
	if err != nil {
		return err
	}
	bcr.Switch.Broadcast(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}
