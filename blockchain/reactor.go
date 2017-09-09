package blockchain

import (
	"bytes"
	"context"
	"reflect"
    "time"
	"net/http"
	"fmt"

	wire "github.com/tendermint/go-wire"
	"github.com/bytom/p2p"
	"github.com/bytom/types"
    "github.com/bytom/protocol/bc/legacy"
    "github.com/bytom/protocol"
	"github.com/bytom/encoding/json"
	cmn "github.com/tendermint/tmlibs/common"
	"github.com/bytom/blockchain/txdb"
	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/txfeed"
	"github.com/bytom/log"
	//"github.com/bytom/net/http/gzip"
	"github.com/bytom/net/http/httpjson"
	//"github.com/bytom/net/http/limit"
	"github.com/bytom/net/http/static"
	"github.com/bytom/generated/dashboard"
	"github.com/bytom/errors"
	"github.com/bytom/blockchain/txbuilder"
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
	crosscoreRPCPrefix = "/rpc/"
)

// BlockchainReactor handles long-term catchup syncing.
type BlockchainReactor struct {
	p2p.BaseReactor

	chain        *protocol.Chain
	store        *txdb.Store
	accounts	 *account.Manager
	assets	     *asset.Registry
	txFeeds		 *txfeed.TxFeed
	pool         *BlockPool
	mux          *http.ServeMux
	handler      http.Handler
	fastSync     bool
	requestsCh   chan BlockRequest
	timeoutsCh   chan string
	submitter    txbuilder.Submitter

	evsw types.EventSwitch
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
	log.Printf(ctx,"creat-block-key")
}

func webAssetsHandler(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", static.Handler{
		Assets:  dashboard.Files,
		Default: "index.html",
	}))
	mux.Handle("/", next)
	return mux
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
	m.Handle("/create-account", jsonHandler(bcr.createAccount))
	m.Handle("/create-asset", jsonHandler(bcr.createAsset))
	m.Handle("/update-account-tags",jsonHandler(bcr.updateAccountTags))
	m.Handle("/update-asset-tags",jsonHandler(bcr.updateAssetTags))
	m.Handle("/build-transaction", jsonHandler(bcr.build))
	m.Handle("/create-control-program",jsonHandler(bcr.createControlProgram))
	m.Handle("/create-account-receiver", jsonHandler(bcr.createAccountReceiver))
	m.Handle("/create-transaction-feed", jsonHandler(bcr.createTxFeed))
	m.Handle("/get-transaction-feed", jsonHandler(bcr.getTxFeed))
	m.Handle("/update-transaction-feed", jsonHandler(bcr.updateTxFeed))
	m.Handle("/delete-transaction-feed", jsonHandler(bcr.deleteTxFeed))
	m.Handle("/", alwaysError(errors.New("not Found")))
	m.Handle("/info", jsonHandler(bcr.info))
	m.Handle("/create-block-key", jsonHandler(bcr.createblockkey))
	m.Handle("/submit-transaction", jsonHandler(bcr.submit))

    latencyHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if l := latency(m, req); l != nil {
			defer l.RecordSince(time.Now())
		}
		m.ServeHTTP(w, req)
		})
	handler := maxBytes(latencyHandler) // TODO(tessr): consider moving this to non-core specific mux
	handler = webAssetsHandler(handler)
/*	handler = healthHandler(handler)
	for _, l := range a.requestLimits {
		handler = limit.Handler(handler, alwaysError(errRateLimited), l.perSecond, l.burst, l.key)
	}
	handler = gzip.Handler{Handler: handler}
	handler = coreCounter(handler)
	handler = timeoutContextHandler(handler)
	if a.config != nil && a.config.BlockchainId != nil {
		handler = blockchainIDHandler(handler, a.config.BlockchainId.String())
	}
	*/
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

func NewBlockchainReactor(store *txdb.Store, chain *protocol.Chain, accounts *account.Manager, assets *asset.Registry, fastSync bool) *BlockchainReactor {
    requestsCh    := make(chan BlockRequest, defaultChannelCapacity)
    timeoutsCh    := make(chan string, defaultChannelCapacity)
    pool := NewBlockPool(
        store.Height()+1,
        requestsCh,
        timeoutsCh,
    )
    bcR := &BlockchainReactor {
        chain:         chain,
        store:         store,
		accounts:      accounts,
		assets:		   assets,
        pool:          pool,
		mux:           http.NewServeMux(),
        fastSync:      fastSync,
        requestsCh:    requestsCh,
        timeoutsCh:   timeoutsCh,
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
	if !peer.Send(BlockchainChannel, struct{ BlockchainMessage }{&bcStatusResponseMessage{bcR.store.Height()}}) {
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
		// Got a request for a block. Respond with block if we have it.
		block, _:= bcR.store.GetBlock(msg.Height)
		if block != nil {
			msg := &bcBlockResponseMessage{Block: block}
			queued := src.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
			if !queued {
				// queue is full, just ignore.
			}
		} else {
			// TODO peer is asking for things we don't have.
		}
	case *bcBlockResponseMessage:
		// Got a block.
		bcR.pool.AddBlock(src.Key, msg.Block, len(msgBytes))
	case *bcStatusRequestMessage:
		// Send peer our state.
		queued := src.TrySend(BlockchainChannel, struct{ BlockchainMessage }{&bcStatusResponseMessage{bcR.store.Height()}})
		if !queued {
			// sorry
		}
	case *bcStatusResponseMessage:
		// Got a peer status. Unverified.
		bcR.pool.SetPeerHeight(src.Key, msg.Height)
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
		case _ = <-statusUpdateTicker.C:
			// ask for status updates
			go bcR.BroadcastStatusRequest()
		/*case _ = <-switchToConsensusTicker.C:
			height, numPending, _ := bcR.pool.GetStatus()
			outbound, inbound, _ := bcR.Switch.NumPeers()
			bcR.Logger.Info("Consensus ticker", "numPending", numPending, "total", len(bcR.pool.requesters),
				"outbound", outbound, "inbound", inbound)
			if bcR.pool.IsCaughtUp() {
				bcR.Logger.Info("Time to switch to consensus reactor!", "height", height)
				bcR.pool.Stop()

				conR := bcR.Switch.Reactor("CONSENSUS").(consensusReactor)
				conR.SwitchToConsensus(bcR.state)

				break FOR_LOOP
			}*/
		case _ = <-trySyncTicker.C: // chan time
			// This loop can be slow as long as it's doing syncing work.
		SYNC_LOOP:
			for i := 0; i < 10; i++ {
				// See if there are any blocks to sync.
				first, second := bcR.pool.PeekTwoBlocks()
				bcR.Logger.Info("TrySync peeked", "first", first, "second", second)
				if first == nil || second == nil {
					// We need both to sync the first block.
					break SYNC_LOOP
				}
			    bcR.pool.PopRequest()
                bcR.store.SaveBlock(first)
			}
			continue FOR_LOOP
		case <-bcR.Quit:
			break FOR_LOOP
		}
	}
}

// BroadcastStatusRequest broadcasts `BlockStore` height.
func (bcR *BlockchainReactor) BroadcastStatusRequest() error {
	bcR.Switch.Broadcast(BlockchainChannel, struct{ BlockchainMessage }{&bcStatusRequestMessage{bcR.store.Height()}})
	return nil
}


/*
// SetEventSwitch implements events.Eventable
func (bcR *BlockchainReactor) SetEventSwitch(evsw types.EventSwitch) {
	bcR.evsw = evsw
}
*/

//-----------------------------------------------------------------------------
// Messages

const (
	msgTypeBlockRequest   = byte(0x10)
	msgTypeBlockResponse  = byte(0x11)
	msgTypeStatusResponse = byte(0x20)
	msgTypeStatusRequest  = byte(0x21)
)

// BlockchainMessage is a generic message for this reactor.
type BlockchainMessage interface{}

var _ = wire.RegisterInterface(
	struct{ BlockchainMessage }{},
	wire.ConcreteType{&bcBlockRequestMessage{}, msgTypeBlockRequest},
	wire.ConcreteType{&bcBlockResponseMessage{}, msgTypeBlockResponse},
	wire.ConcreteType{&bcStatusResponseMessage{}, msgTypeStatusResponse},
	wire.ConcreteType{&bcStatusRequestMessage{}, msgTypeStatusRequest},
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

// NOTE: keep up-to-date with maxBlockchainResponseSize
type bcBlockResponseMessage struct {
	Block *legacy.Block
}

func (m *bcBlockResponseMessage) String() string {
	return cmn.Fmt("[bcBlockResponseMessage %v]", m.Block.Height)
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
