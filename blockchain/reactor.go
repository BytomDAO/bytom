package blockchain

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/bytom/blockchain/accesstoken"
	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/rpc"
	ctypes "github.com/bytom/blockchain/rpc/types"
	"github.com/bytom/blockchain/txfeed"
	"github.com/bytom/blockchain/wallet"
	"github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/mining/cpuminer"
	"github.com/bytom/net/http/httpjson"
	"github.com/bytom/p2p"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/validation"
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

type Response struct {
	Status string
	Msg    string
	Data   []string
}

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

const (
	SUCCESS = "success"
	FAIL    = "fail"
	ERROR   = "error"
)

// DefaultRawResponse is used as the default response when fail to get data
var DefaultRawResponse = []byte(`{"Status":"error","Msg":"Unable to get data","Data":null}`)

// Response describes the response standard.
type Response struct {
	Status string   `json:"status"`
	Msg    string   `json:"msg"`
	Data   []string `json:"data"`
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

func (bcr *BlockchainReactor) BuildHander() {
	m := bcr.mux
	if bcr.accounts != nil && bcr.assets != nil {
		m.Handle("/create-account", jsonHandler(bcr.createAccount))
		m.Handle("/update-account-tags", jsonHandler(bcr.updateAccountTags))
		m.Handle("/create-account-receiver", jsonHandler(bcr.createAccountReceiver))
		m.Handle("/list-accounts", jsonHandler(bcr.listAccounts))
		m.Handle("/create-asset", jsonHandler(bcr.createAsset))
		m.Handle("/update-asset-tags", jsonHandler(bcr.updateAssetTags))
		m.Handle("/list-assets", jsonHandler(bcr.listAssets))
		m.Handle("/list-transactions", jsonHandler(bcr.listTransactions))
		m.Handle("/list-balances", jsonHandler(bcr.listBalances))
	} else {
		log.Warn("Please enable wallet")
	}

	m.Handle("/build-transaction", jsonHandler(bcr.build))
	m.Handle("/create-control-program", jsonHandler(bcr.createControlProgram))
	m.Handle("/create-transaction-feed", jsonHandler(bcr.createTxFeed))
	m.Handle("/get-transaction-feed", jsonHandler(bcr.getTxFeed))
	m.Handle("/update-transaction-feed", jsonHandler(bcr.updateTxFeed))
	m.Handle("/delete-transaction-feed", jsonHandler(bcr.deleteTxFeed))
	m.Handle("/list-transaction-feeds", jsonHandler(bcr.listTxFeeds))
	m.Handle("/list-unspent-outputs", jsonHandler(bcr.listUnspentOutputs))
	m.Handle("/", alwaysError(errors.New("not Found")))
	m.Handle("/info", jsonHandler(bcr.info))
	m.Handle("/submit-transaction", jsonHandler(bcr.submit))
	m.Handle("/create-access-token", jsonHandler(bcr.createAccessToken))
	m.Handle("/list-access-token", jsonHandler(bcr.listAccessTokens))
	m.Handle("/delete-access-token", jsonHandler(bcr.deleteAccessToken))
	m.Handle("/check-access-token", jsonHandler(bcr.checkAccessToken))

	//hsm api
	m.Handle("/create-key", jsonHandler(bcr.pseudohsmCreateKey))
	m.Handle("/list-keys", jsonHandler(bcr.pseudohsmListKeys))
	m.Handle("/delete-key", jsonHandler(bcr.pseudohsmDeleteKey))
	m.Handle("/sign-transactions", jsonHandler(bcr.pseudohsmSignTemplates))
	m.Handle("/reset-password", jsonHandler(bcr.pseudohsmResetPassword))
	m.Handle("/net-info", jsonHandler(bcr.getNetInfo))
	m.Handle("/get-best-block-hash", jsonHandler(bcr.getBestBlockHash))
	m.Handle("/get-block-header-by-hash", jsonHandler(bcr.getBlockHeaderByHash))
	m.Handle("/get-block-transactions-count-by-hash", jsonHandler(bcr.getBlockTransactionsCountByHash))
	m.Handle("/get-block-by-hash", jsonHandler(bcr.getBlockByHash))
	m.Handle("/net-listening", jsonHandler(bcr.isNetListening))
	m.Handle("/net-syncing", jsonHandler(bcr.isNetSyncing))
	m.Handle("/peer-count", jsonHandler(bcr.peerCount))
	m.Handle("/get-block-by-height", jsonHandler(bcr.getBlockByHeight))
	m.Handle("/get-block-transactions-count-by-height", jsonHandler(bcr.getBlockTransactionsCountByHeight))
	m.Handle("/block-height", jsonHandler(bcr.blockHeight))
	m.Handle("/is-mining", jsonHandler(bcr.isMining))
	m.Handle("/gas-rate", jsonHandler(bcr.gasRate))

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

func (bcR *BlockchainReactor) getNetInfo() (*ctypes.ResultNetInfo, error) {
	return rpc.NetInfo(bcR.sw)
}

func (bcR *BlockchainReactor) getBestBlockHash() *bc.Hash {
	return bcR.chain.BestBlockHash()
}

func (bcr *BlockchainReactor) getBlockHeaderByHash(strHash string) string {
	var buf bytes.Buffer
	hash := bc.Hash{}
	if err := hash.UnmarshalText([]byte(strHash)); err != nil {
		log.WithField("error", err).Error("Error occurs when transforming string hash to hash struct")
	}
	block, err := bcr.chain.GetBlockByHash(&hash)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return ""
	}
	bcBlock := legacy.MapBlock(block)
	header, _ := stdjson.MarshalIndent(bcBlock.BlockHeader, "", "  ")
	buf.WriteString(string(header))
	return buf.String()
}

type TxJSON struct {
	Inputs  []bc.Entry `json:"inputs"`
	Outputs []bc.Entry `json:"outputs"`
}

type GetBlockByHashJSON struct {
	BlockHeader  *bc.BlockHeader `json:"block_header"`
	Transactions []*TxJSON       `json:"transactions"`
}

func (bcr *BlockchainReactor) getBlockByHash(strHash string) string {
	hash := bc.Hash{}
	if err := hash.UnmarshalText([]byte(strHash)); err != nil {
		log.WithField("error", err).Error("Error occurs when transforming string hash to hash struct")
		return err.Error()
	}

	legacyBlock, err := bcr.chain.GetBlockByHash(&hash)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return err.Error()
	}

	bcBlock := legacy.MapBlock(legacyBlock)
	res := &GetBlockByHashJSON{BlockHeader: bcBlock.BlockHeader}
	for _, tx := range bcBlock.Transactions {
		txJSON := &TxJSON{}
		for _, e := range tx.Entries {
			switch e := e.(type) {
			case *bc.Issuance:
				txJSON.Inputs = append(txJSON.Inputs, e)
			case *bc.Spend:
				txJSON.Inputs = append(txJSON.Inputs, e)
			case *bc.Retirement:
				txJSON.Outputs = append(txJSON.Outputs, e)
			case *bc.Output:
				txJSON.Outputs = append(txJSON.Outputs, e)
			default:
				continue
			}
		}
		res.Transactions = append(res.Transactions, txJSON)
	}

	ret, err := stdjson.Marshal(res)
	if err != nil {
		return err.Error()
	}
	return string(ret)
}

func (bcr *BlockchainReactor) getBlockByHeight(height uint64) []byte {
	legacyBlock, err := bcr.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return DefaultRawResponse
	}

	bcBlock := legacy.MapBlock(legacyBlock)
	res := &GetBlockByHashJSON{BlockHeader: bcBlock.BlockHeader}
	for _, tx := range bcBlock.Transactions {
		txJSON := &TxJSON{}
		for _, e := range tx.Entries {
			switch e := e.(type) {
			case *bc.Issuance:
				txJSON.Inputs = append(txJSON.Inputs, e)
			case *bc.Spend:
				txJSON.Inputs = append(txJSON.Inputs, e)
			case *bc.Retirement:
				txJSON.Outputs = append(txJSON.Outputs, e)
			case *bc.Output:
				txJSON.Outputs = append(txJSON.Outputs, e)
			default:
				continue
			}
		}
		res.Transactions = append(res.Transactions, txJSON)
	}

	ret, err := stdjson.Marshal(res)
	if err != nil {
		return DefaultRawResponse
	}
	data := []string{string(ret)}
	return resWrapper(data)
}

func (bcr *BlockchainReactor) getBlockTransactionsCountByHash(strHash string) (int, error) {
	hash := bc.Hash{}
	if err := hash.UnmarshalText([]byte(strHash)); err != nil {
		log.WithField("error", err).Error("Error occurs when transforming string hash to hash struct")
		return -1, err
	}

	legacyBlock, err := bcr.chain.GetBlockByHash(&hash)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return -1, err
	}
	return len(legacyBlock.Transactions), nil
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

func (bcr *BlockchainReactor) isNetListening() bool {
	return bcr.sw.IsListening()
}

func (bcr *BlockchainReactor) peerCount() int {
	return len(bcr.sw.Peers().List())
}

func (bcr *BlockchainReactor) isNetSyncing() bool {
	return bcr.blockKeeper.IsCaughtUp()
}

func (bcr *BlockchainReactor) getBlockTransactionsCountByHeight(height uint64) []byte {
	legacyBlock, err := bcr.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return DefaultRawResponse
	}
	data := []string{strconv.FormatInt(int64(len(legacyBlock.Transactions)), 16)}
	log.Infof("%v", data)
	return resWrapper(data)
}

func (bcr *BlockchainReactor) blockHeight() []byte {
	data := []string{strconv.FormatUint(bcr.chain.Height(), 16)}
	return resWrapper(data)
}

func (bcr *BlockchainReactor) isMining() []byte {
	data := []string{strconv.FormatBool(bcr.mining.IsMining())}
	return resWrapper(data)
}

func (bcr *BlockchainReactor) gasRate() []byte {
	data := []string{strconv.FormatInt(validation.GasRate, 16)}
	return resWrapper(data)
}

func resWrapper(data []string) []byte {
	response := Response{Status: SUCCESS, Data: data}
	rawResponse, err := stdjson.Marshal(response)
	if err != nil {
		return DefaultRawResponse
	}
	return rawResponse
}
