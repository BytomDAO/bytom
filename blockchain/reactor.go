package blockchain

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/bytom/blockchain/accesstoken"
	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txdb"
	"github.com/bytom/blockchain/txfeed"
	"github.com/bytom/encoding/json"
	"github.com/bytom/log"
	"github.com/bytom/mining/cpuminer"
	"github.com/bytom/net/http/httpjson"
	"github.com/bytom/p2p"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/types"
	wire "github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/bytom/blockchain/pin"
	"github.com/bytom/errors"
)

const (
	// BlockchainChannel is a channel for blocks and status updates (`BlockStore` height)
	BlockchainChannel = byte(0x40)

	defaultChannelCapacity = 100
	defaultSleepIntervalMS = 500
	trySyncIntervalMS      = 100
	// stop syncing when last block's time is
	// within this much of the system time.
	// stopSyncingDurationMinutes = 10

	// ask for best height every 10s
	statusUpdateIntervalSeconds = 10
	// check if we should switch to consensus reactor
	switchToConsensusIntervalSeconds = 1
	maxBlockchainResponseSize        = 22020096 + 2
	crosscoreRPCPrefix               = "/rpc/"
)

// BlockchainReactor handles long-term catchup syncing.
type BlockchainReactor struct {
	p2p.BaseReactor

	chain       *protocol.Chain
	store       *txdb.Store
	pinStore    *pin.Store
	accounts    *account.Manager
	assets      *asset.Registry
	accesstoken *accesstoken.Token
	txFeeds     *txfeed.TxFeed
	pool        *BlockPool
	txPool      *protocol.TxPool
	hsm         *pseudohsm.HSM
	mining      *cpuminer.CPUMiner
	mux         *http.ServeMux
	handler     http.Handler
	fastSync    bool
	requestsCh  chan BlockRequest
	timeoutsCh  chan string
	evsw        types.EventSwitch
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
		errorFormatter.Log(ctx, err)
		*v = errorFormatter.Format(err)
	}
}

func jsonHandler(f interface{}) http.Handler {
	h, err := httpjson.Handler(f, errorFormatter.Write)
	if err != nil {
		panic(err)
	}
	return h
}

func alwaysError(err error) http.Handler {
	return jsonHandler(func() error { return err })
}

func (bcr *BlockchainReactor) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	bcr.handler.ServeHTTP(rw, req)
}

func (bcr *BlockchainReactor) info(ctx context.Context) (map[string]interface{}, error) {
	//if a.config == nil {
	// never configured
	log.Printf(ctx, "-------info-----")
	return map[string]interface{}{
		"is_configured": false,
		"version":       "0.001",
		"build_commit":  "----",
		"build_date":    "------",
		"build_config":  "---------",
	}, nil
	//}
}

func (bcr *BlockchainReactor) createblockkey(ctx context.Context) {
	log.Printf(ctx, "creat-block-key")
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

func (bcr *BlockchainReactor) BuildHander() {
	m := bcr.mux
	if bcr.accounts != nil {
		m.Handle("/create-account", jsonHandler(bcr.createAccount))
		m.Handle("/update-account-tags", jsonHandler(bcr.updateAccountTags))
		m.Handle("/create-account-receiver", jsonHandler(bcr.createAccountReceiver))
		m.Handle("/list-accounts", jsonHandler(bcr.listAccounts))
	} else {
		log.Printf(context.Background(), "Warning: Please enable wallet")
	}

	if bcr.assets != nil {
		m.Handle("/create-asset", jsonHandler(bcr.createAsset))
		m.Handle("/update-asset-tags", jsonHandler(bcr.updateAssetTags))
		m.Handle("/list-assets", jsonHandler(bcr.listAssets))
	} else {
		log.Printf(context.Background(), "Warning: Please enable wallet")
	}
	m.Handle("/build-transaction", jsonHandler(bcr.build))
	m.Handle("/create-control-program", jsonHandler(bcr.createControlProgram))
	m.Handle("/create-transaction-feed", jsonHandler(bcr.createTxFeed))
	m.Handle("/get-transaction-feed", jsonHandler(bcr.getTxFeed))
	m.Handle("/update-transaction-feed", jsonHandler(bcr.updateTxFeed))
	m.Handle("/delete-transaction-feed", jsonHandler(bcr.deleteTxFeed))
	m.Handle("/list-transaction-feeds", jsonHandler(bcr.listTxFeeds))
	m.Handle("/list-transactions", jsonHandler(bcr.listTransactions))
	m.Handle("/list-balances", jsonHandler(bcr.listBalances))
	m.Handle("/list-unspent-outputs", jsonHandler(bcr.listUnspentOutputs))
	m.Handle("/", alwaysError(errors.New("not Found")))
	m.Handle("/info", jsonHandler(bcr.info))
	m.Handle("/create-block-key", jsonHandler(bcr.createblockkey))
	m.Handle("/submit-transaction", jsonHandler(bcr.submit))
	m.Handle("/create-access-token", jsonHandler(bcr.createAccessToken))
	m.Handle("/list-access-tokens", jsonHandler(bcr.listAccessTokens))
	m.Handle("/delete-access-token", jsonHandler(bcr.deleteAccessToken))
	//hsm api
	m.Handle("/create-key", jsonHandler(bcr.pseudohsmCreateKey))
	m.Handle("/list-keys", jsonHandler(bcr.pseudohsmListKeys))
	m.Handle("/delete-key", jsonHandler(bcr.pseudohsmDeleteKey))
	m.Handle("/sign-transactions", jsonHandler(bcr.pseudohsmSignTemplates))
	m.Handle("/reset-password", jsonHandler(bcr.pseudohsmResetPassword))
	m.Handle("/update-alias", jsonHandler(bcr.pseudohsmUpdateAlias))

	latencyHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if l := latency(m, req); l != nil {
			defer l.RecordSince(time.Now())
		}
		m.ServeHTTP(w, req)
	})
	handler := maxBytes(latencyHandler) // TODO(tessr): consider moving this to non-core specific mux

	bcr.handler = handler
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

func NewBlockchainReactor(store *txdb.Store,
	chain *protocol.Chain,
	txPool *protocol.TxPool,
	accounts *account.Manager,
	assets *asset.Registry,
	hsm *pseudohsm.HSM,
	fastSync bool,
	pinStore *pin.Store) *BlockchainReactor {
	requestsCh := make(chan BlockRequest, defaultChannelCapacity)
	timeoutsCh := make(chan string, defaultChannelCapacity)
	pool := NewBlockPool(
		chain.Height()+1,
		requestsCh,
		timeoutsCh,
	)
	mining := cpuminer.NewCPUMiner(chain, txPool)
	bcR := &BlockchainReactor{
		chain:      chain,
		store:      store,
		pinStore:   pinStore,
		accounts:   accounts,
		assets:     assets,
		pool:       pool,
		txPool:     txPool,
		mining:     mining,
		mux:        http.NewServeMux(),
		hsm:        hsm,
		fastSync:   fastSync,
		requestsCh: requestsCh,
		timeoutsCh: timeoutsCh,
	}
	bcR.BaseReactor = *p2p.NewBaseReactor("BlockchainReactor", bcR)
	return bcR
}

// OnStart implements BaseService
func (bcR *BlockchainReactor) OnStart() error {
	bcR.BaseReactor.OnStart()
	bcR.BuildHander()
	if bcR.fastSync {
		_, err := bcR.pool.Start()
		if err != nil {
			return err
		}
		go bcR.poolRoutine()
	}
	return nil
}

// OnStop implements BaseService
func (bcR *BlockchainReactor) OnStop() {
	bcR.BaseReactor.OnStop()
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
	if !peer.Send(BlockchainChannel, struct{ BlockchainMessage }{&bcStatusResponseMessage{bcR.chain.Height()}}) {
		// doing nothing, will try later in `poolRoutine`
	}
}

// RemovePeer implements Reactor by removing peer from the pool.
func (bcR *BlockchainReactor) RemovePeer(peer *p2p.Peer, reason interface{}) {
	bcR.pool.RemovePeer(peer.Key)
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (bcR *BlockchainReactor) Receive(chID byte, src *p2p.Peer, msgBytes []byte) {
	_, msg, err := DecodeMessage(msgBytes)
	if err != nil {
		bcR.Logger.Error("Error decoding message", "error", err)
		return
	}

	bcR.Logger.Info("Receive", "src", src, "chID", chID, "msg", msg)

	switch msg := msg.(type) {
	case *bcBlockRequestMessage:
		block, err := bcR.chain.GetBlockByHeight(msg.Height)
		if err != nil {
			bcR.Logger.Info("skip sent the block response due to block is nil")
			return
		}
		rawBlock, err := block.MarshalText()
		if err == nil {
			msg := &bcBlockResponseMessage{RawBlock: rawBlock}
			src.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
		}
	case *bcBlockResponseMessage:
		// Got a block.
		bcR.pool.AddBlock(src.Key, msg.GetBlock(), len(msgBytes))
	case *bcStatusRequestMessage:
		// Send peer our state.
		queued := src.TrySend(BlockchainChannel, struct{ BlockchainMessage }{&bcStatusResponseMessage{bcR.chain.Height()}})
		if !queued {
			// sorry
		}
	case *bcStatusResponseMessage:
		// Got a peer status. Unverified.
		bcR.pool.SetPeerHeight(src.Key, msg.Height)
	case *bcTransactionMessage:
		tx := msg.GetTransaction()

		if err := bcR.chain.ValidateTx(tx); err != nil {
			bcR.Logger.Error("fail to sync transaction to txPool", "err", err)
		}
	default:
		bcR.Logger.Error(cmn.Fmt("Unknown message type %v", reflect.TypeOf(msg)))
	}
}

// Handle messages from the poolReactor telling the reactor what to do.
// NOTE: Don't sleep in the FOR_LOOP or otherwise slow it down!
// (Except for the SYNC_LOOP, which is the primary purpose and must be synchronous.)
func (bcR *BlockchainReactor) poolRoutine() {

	trySyncTicker := time.NewTicker(trySyncIntervalMS * time.Millisecond)
	statusUpdateTicker := time.NewTicker(statusUpdateIntervalSeconds * time.Second)
	newTxCh := bcR.txPool.GetNewTxCh()
	//switchToConsensusTicker := time.NewTicker(switchToConsensusIntervalSeconds * time.Second)

FOR_LOOP:
	for {

		select {
		case request := <-bcR.requestsCh: // chan BlockRequest
			peer := bcR.Switch.Peers().Get(request.PeerID)
			if peer == nil {
				continue FOR_LOOP // Peer has since been disconnected.
			}
			msg := &bcBlockRequestMessage{request.Height}
			queued := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
			if !queued {
				// We couldn't make the request, send-queue full.
				// The pool handles timeouts, just let it go.
				continue FOR_LOOP
			}
		case peerID := <-bcR.timeoutsCh: // chan string
			// Peer timed out.
			peer := bcR.Switch.Peers().Get(peerID)
			if peer != nil {
				bcR.Switch.StopPeerForError(peer, errors.New("BlockchainReactor Timeout"))
			}
		case newTx := <-newTxCh:
			go bcR.BroadcastTransaction(newTx)
		case _ = <-statusUpdateTicker.C:
			// ask for status updates
			go bcR.BroadcastStatusRequest()
		case _ = <-trySyncTicker.C: // chan time
		SYNC_LOOP:
			for i := 0; i < 10; i++ {
				// See if there are any blocks to sync.
				block, peerID := bcR.pool.PeekBlock()
				if block == nil {
					break SYNC_LOOP
				}
				bcR.pool.PopRequest()

				isOrphan, err := bcR.chain.ProcessBlock(block)
				if err != nil {
					bcR.Logger.Info("fail to sync commit block", "blockHeigh", block.BlockHeader.Height, "error", err)
				}

				if isOrphan {
					src := bcR.Switch.Peers().Get(peerID)
					if src == nil {
						continue
					}
					src.TrySend(BlockchainChannel, struct{ BlockchainMessage }{&bcBlockRequestMessage{Height: block.Height - 1}})
				}
			}
			continue FOR_LOOP
		case <-bcR.Quit:
			break FOR_LOOP
		}
		if bcR.pool.IsCaughtUp() && !bcR.mining.IsMining() {
			bcR.Logger.Info("start to mining")
			bcR.mining.Start()
		}
	}
}

// BroadcastStatusRequest broadcasts `BlockStore` height.
func (bcR *BlockchainReactor) BroadcastStatusRequest() error {
	bcR.Switch.Broadcast(BlockchainChannel, struct{ BlockchainMessage }{&bcStatusRequestMessage{bcR.chain.Height()}})
	return nil
}

func (bcR *BlockchainReactor) BroadcastTransaction(tx *legacy.Tx) error {
	rawTx, err := tx.TxData.MarshalText()
	if err != nil {
		return err
	}
	bcR.Switch.Broadcast(BlockchainChannel, struct{ BlockchainMessage }{&bcTransactionMessage{rawTx}})
	return nil
}

//-----------------------------------------------------------------------------
// Messages

const (
	msgTypeBlockRequest       = byte(0x10)
	msgTypeBlockResponse      = byte(0x11)
	msgTypeStatusResponse     = byte(0x20)
	msgTypeStatusRequest      = byte(0x21)
	msgTypeTransactionRequest = byte(0x30)
)

// BlockchainMessage is a generic message for this reactor.
type BlockchainMessage interface{}

var _ = wire.RegisterInterface(
	struct{ BlockchainMessage }{},
	wire.ConcreteType{&bcBlockRequestMessage{}, msgTypeBlockRequest},
	wire.ConcreteType{&bcBlockResponseMessage{}, msgTypeBlockResponse},
	wire.ConcreteType{&bcStatusResponseMessage{}, msgTypeStatusResponse},
	wire.ConcreteType{&bcStatusRequestMessage{}, msgTypeStatusRequest},
	wire.ConcreteType{&bcTransactionMessage{}, msgTypeTransactionRequest},
)

// DecodeMessage decodes BlockchainMessage.
// TODO: ensure that bz is completely read.
func DecodeMessage(bz []byte) (msgType byte, msg BlockchainMessage, err error) {
	msgType = bz[0]
	n := int(0)
	r := bytes.NewReader(bz)
	msg = wire.ReadBinary(struct{ BlockchainMessage }{}, r, maxBlockchainResponseSize, &n, &err).(struct{ BlockchainMessage }).BlockchainMessage
	if err != nil && n != len(bz) {
		err = errors.New("DecodeMessage() had bytes left over")
	}
	return
}

//-----------------------------------

type bcBlockRequestMessage struct {
	Height uint64
}

func (m *bcBlockRequestMessage) String() string {
	return cmn.Fmt("[bcBlockRequestMessage %v]", m.Height)
}

//-------------------------------------

type bcTransactionMessage struct {
	RawTx []byte
}

func (m *bcTransactionMessage) GetTransaction() *legacy.Tx {
	tx := &legacy.Tx{}
	tx.UnmarshalText(m.RawTx)
	return tx
}

//-------------------------------------

//-------------------------------------

// NOTE: keep up-to-date with maxBlockchainResponseSize
type bcBlockResponseMessage struct {
	RawBlock []byte
}

func (m *bcBlockResponseMessage) GetBlock() *legacy.Block {
	block := &legacy.Block{
		BlockHeader:  legacy.BlockHeader{},
		Transactions: []*legacy.Tx{},
	}
	block.UnmarshalText(m.RawBlock)
	return block
}

func (m *bcBlockResponseMessage) String() string {
	block := m.GetBlock()
	return cmn.Fmt("[bcBlockResponseMessage %v]", block.BlockHeader.Height)
}

//-------------------------------------

type bcStatusRequestMessage struct {
	Height uint64
}

func (m *bcStatusRequestMessage) String() string {
	return cmn.Fmt("[bcStatusRequestMessage %v]", m.Height)
}

//-------------------------------------

type bcStatusResponseMessage struct {
	Height uint64
}

func (m *bcStatusResponseMessage) String() string {
	return cmn.Fmt("[bcStatusResponseMessage %v]", m.Height)
}
