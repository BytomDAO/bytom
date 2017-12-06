package blockchain

import (
	"fmt"
	"time"
	"context"
	"reflect"
	"net/http"

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
	"github.com/bytom/p2p"
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
	SUCCESS = "success"
	FAIL    = "fail"
	ERROR   = "error"
)

// Response describes the response standard.
type Response struct {
	Status string
	Msg    string
	Data   []string
}

// DefaultRawResponse is used as the default response when fail to get data.
var DefaultRawResponse = []byte(`{"Status":"error","Msg":"Unable to get data","Data":null}`)

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
	mux           *http.ServeMux
	sw            *p2p.Switch
	handler       http.Handler
	evsw          types.EventSwitch
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
}

func NewBlockchainReactor(chain *protocol.Chain, txPool *protocol.TxPool, accounts *account.Manager, assets *asset.Registry, sw *p2p.Switch, hsm *pseudohsm.HSM, wallet *wallet.Wallet, txfeeds *txfeed.Tracker, accessTokens *accesstoken.CredentialStore) *BlockchainReactor {
	mining := cpuminer.NewCPUMiner(chain, accounts, txPool)
	bcR := &BlockchainReactor{
		chain:         chain,
		wallet:        wallet,
		accounts:      accounts,
		assets:        assets,
		blockKeeper:   newBlockKeeper(chain, sw),
		txPool:        txPool,
		mining:        mining,
		mux:           http.NewServeMux(),
		sw:            sw,
		hsm:           hsm,
		txFeedTracker: txfeeds,
		accessTokens:  accessTokens,
	}
	bcR.BaseReactor = *p2p.NewBaseReactor("BlockchainReactor", bcR)
	return bcR
}

// OnStart implements BaseService
func (bcR *BlockchainReactor) OnStart() error {
	bcR.BaseReactor.OnStart()
	bcR.BuildHander()

	bcR.mining.Start()
	go bcR.syncRoutine()
	return nil
}

// OnStop implements BaseService
func (bcR *BlockchainReactor) OnStop() {
	bcR.BaseReactor.OnStop()
	bcR.mining.Stop()
}

// GetChannels implements Reactor
func (bcR *BlockchainReactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		&p2p.ChannelDescriptor{
			ID:                BlockchainChannel,
			Priority:          5,
			SendQueueCapacity: 100,
		},
	}
}

// AddPeer implements Reactor by sending our state to peer.
func (bcR *BlockchainReactor) AddPeer(peer *p2p.Peer) {
	peer.Send(BlockchainChannel, struct{ BlockchainMessage }{&StatusRequestMessage{}})
}

// RemovePeer implements Reactor by removing peer from the pool.
func (bcR *BlockchainReactor) RemovePeer(peer *p2p.Peer, reason interface{}) {
	bcR.blockKeeper.RemovePeer(peer.Key)
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (bcR *BlockchainReactor) Receive(chID byte, src *p2p.Peer, msgBytes []byte) {
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
			block, err = bcR.chain.GetBlockByHeight(msg.Height)
		} else {
			block, err = bcR.chain.GetBlockByHash(msg.GetHash())
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
		bcR.blockKeeper.AddBlock(msg.GetBlock(), src.Key)

	case *StatusRequestMessage:
		block, _ := bcR.chain.State()
		src.TrySend(BlockchainChannel, struct{ BlockchainMessage }{NewStatusResponseMessage(block)})

	case *StatusResponseMessage:
		bcR.blockKeeper.SetPeerHeight(src.Key, msg.Height, msg.GetHash())

	case *TransactionNotifyMessage:
		tx := msg.GetTransaction()
		if err := bcR.chain.ValidateTx(tx); err != nil {
			log.Errorf("TransactionNotifyMessage: %v", err)
		}

	default:
		log.Error(cmn.Fmt("Unknown message type %v", reflect.TypeOf(msg)))
	}
}

// Handle messages from the poolReactor telling the reactor what to do.
// NOTE: Don't sleep in the FOR_LOOP or otherwise slow it down!
// (Except for the SYNC_LOOP, which is the primary purpose and must be synchronous.)
func (bcR *BlockchainReactor) syncRoutine() {
	statusUpdateTicker := time.NewTicker(statusUpdateIntervalSeconds * time.Second)
	newTxCh := bcR.txPool.GetNewTxCh()

	for {
		select {
		case newTx := <-newTxCh:
			bcR.txFeedTracker.TxFilter(newTx)
			go bcR.BroadcastTransaction(newTx)
		case _ = <-statusUpdateTicker.C:
			go bcR.BroadcastStatusResponse()

			// mining if and only if block sync is finished
			if bcR.blockKeeper.IsCaughtUp() {
				bcR.mining.Start()
			} else {
				bcR.mining.Stop()
			}
		case <-bcR.Quit:
			return
		}
	}
}


// BroadcastStatusRequest broadcasts `BlockStore` height.
func (bcR *BlockchainReactor) BroadcastStatusResponse() {
	block, _ := bcR.chain.State()
	bcR.Switch.Broadcast(BlockchainChannel, struct{ BlockchainMessage }{NewStatusResponseMessage(block)})
}

func (bcR *BlockchainReactor) BroadcastTransaction(tx *legacy.Tx) error {
	msg, err := NewTransactionNotifyMessage(tx)
	if err != nil {
		return err
	}
	bcR.Switch.Broadcast(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}
