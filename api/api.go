package api

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/kr/secureheader"
	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/bytom/accesstoken"
	"github.com/bytom/blockchain/txfeed"
	cfg "github.com/bytom/config"
	"github.com/bytom/dashboard/dashboard"
	"github.com/bytom/dashboard/equity"
	"github.com/bytom/errors"
	"github.com/bytom/mining/cpuminer"
	"github.com/bytom/mining/miningpool"
	"github.com/bytom/net/http/authn"
	"github.com/bytom/net/http/gzip"
	"github.com/bytom/net/http/httpjson"
	"github.com/bytom/net/http/static"
	"github.com/bytom/net/websocket"
	"github.com/bytom/netsync"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/wallet"
)

var (
	errNotAuthenticated = errors.New("not authenticated")
	httpReadTimeout     = 2 * time.Minute
	httpWriteTimeout    = time.Hour
)

const (
	// SUCCESS indicates the rpc calling is successful.
	SUCCESS = "success"
	// FAIL indicated the rpc calling is failed.
	FAIL = "fail"
)

// Response describes the response standard.
type Response struct {
	Status      string      `json:"status,omitempty"`
	Code        string      `json:"code,omitempty"`
	Msg         string      `json:"msg,omitempty"`
	ErrorDetail string      `json:"error_detail,omitempty"`
	Data        interface{} `json:"data,omitempty"`
}

//NewSuccessResponse success response
func NewSuccessResponse(data interface{}) Response {
	return Response{Status: SUCCESS, Data: data}
}

//FormatErrResp format error response
func FormatErrResp(err error) (response Response) {
	response = Response{Status: FAIL}
	root := errors.Root(err)
	// Some types cannot be used as map keys, for example slices.
	// If an error's underlying type is one of these, don't panic.
	// Just treat it like any other missing entry.
	defer func() {
		if err := recover(); err != nil {
			response.ErrorDetail = ""
		}
	}()

	if info, ok := respErrFormatter[root]; ok {
		response.Code = info.ChainCode
		response.Msg = info.Message
		response.ErrorDetail = err.Error()
	} else {
		response.Code = respErrFormatter[ErrDefault].ChainCode
		response.Msg = respErrFormatter[ErrDefault].Message
		response.ErrorDetail = err.Error()
	}
	return response
}

//NewErrorResponse error response
func NewErrorResponse(err error) Response {
	response := FormatErrResp(err)
	return response
}

type waitHandler struct {
	h  http.Handler
	wg sync.WaitGroup
}

func (wh *waitHandler) Set(h http.Handler) {
	wh.h = h
	wh.wg.Done()
}

func (wh *waitHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	wh.wg.Wait()
	wh.h.ServeHTTP(w, req)
}

// API is the scheduling center for server
type API struct {
	sync            *netsync.SyncManager
	wallet          *wallet.Wallet
	accessTokens    *accesstoken.CredentialStore
	chain           *protocol.Chain
	server          *http.Server
	handler         http.Handler
	txFeedTracker   *txfeed.Tracker
	cpuMiner        *cpuminer.CPUMiner
	miningPool      *miningpool.MiningPool
	notificationMgr *websocket.WSNotificationManager
	newBlockCh      chan *bc.Hash
}

func (a *API) initServer(config *cfg.Config) {
	// The waitHandler accepts incoming requests, but blocks until its underlying
	// handler is set, when the second phase is complete.
	var coreHandler waitHandler
	var handler http.Handler

	coreHandler.wg.Add(1)
	mux := http.NewServeMux()
	mux.Handle("/", &coreHandler)

	handler = AuthHandler(mux, a.accessTokens, config.Auth.Disable)
	handler = RedirectHandler(handler)

	secureheader.DefaultConfig.PermitClearLoopback = true
	secureheader.DefaultConfig.HTTPSRedirect = false
	secureheader.DefaultConfig.Next = handler

	a.server = &http.Server{
		// Note: we should not set TLSConfig here;
		// we took care of TLS with the listener in maybeUseTLS.
		Handler:      secureheader.DefaultConfig,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		// Disable HTTP/2 for now until the Go implementation is more stable.
		// https://github.com/golang/go/issues/16450
		// https://github.com/golang/go/issues/17071
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}

	coreHandler.Set(a)
}

// StartServer start the server
func (a *API) StartServer(address string) {
	log.WithField("api address:", address).Info("Rpc listen")
	listener, err := net.Listen("tcp", address)
	if err != nil {
		cmn.Exit(cmn.Fmt("Failed to register tcp port: %v", err))
	}

	// The `Serve` call has to happen in its own goroutine because
	// it's blocking and we need to proceed to the rest of the core setup after
	// we call it.
	go func() {
		if err := a.server.Serve(listener); err != nil {
			log.WithField("error", errors.Wrap(err, "Serve")).Error("Rpc server")
		}
	}()
}

// NewAPI create and initialize the API
func NewAPI(sync *netsync.SyncManager, wallet *wallet.Wallet, txfeeds *txfeed.Tracker, cpuMiner *cpuminer.CPUMiner, miningPool *miningpool.MiningPool, chain *protocol.Chain, config *cfg.Config, token *accesstoken.CredentialStore, newBlockCh chan *bc.Hash, notificationMgr *websocket.WSNotificationManager) *API {
	api := &API{
		sync:          sync,
		wallet:        wallet,
		chain:         chain,
		accessTokens:  token,
		txFeedTracker: txfeeds,
		cpuMiner:      cpuMiner,
		miningPool:    miningPool,

		newBlockCh:      newBlockCh,
		notificationMgr: notificationMgr,
	}
	api.buildHandler()
	api.initServer(config)

	return api
}

func (a *API) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	a.handler.ServeHTTP(rw, req)
}

// buildHandler is in charge of all the rpc handling.
func (a *API) buildHandler() {
	walletEnable := false
	m := http.NewServeMux()
	if a.wallet != nil {
		walletEnable = true

		m.Handle("/create-account", jsonHandler(a.createAccount))
		m.Handle("/update-account-alias", jsonHandler(a.updateAccountAlias))
		m.Handle("/list-accounts", jsonHandler(a.listAccounts))
		m.Handle("/delete-account", jsonHandler(a.deleteAccount))

		m.Handle("/create-account-receiver", jsonHandler(a.createAccountReceiver))
		m.Handle("/list-addresses", jsonHandler(a.listAddresses))
		m.Handle("/validate-address", jsonHandler(a.validateAddress))
		m.Handle("/list-pubkeys", jsonHandler(a.listPubKeys))

		m.Handle("/get-mining-address", jsonHandler(a.getMiningAddress))
		m.Handle("/set-mining-address", jsonHandler(a.setMiningAddress))

		m.Handle("/get-coinbase-arbitrary", jsonHandler(a.getCoinbaseArbitrary))
		m.Handle("/set-coinbase-arbitrary", jsonHandler(a.setCoinbaseArbitrary))

		m.Handle("/create-asset", jsonHandler(a.createAsset))
		m.Handle("/update-asset-alias", jsonHandler(a.updateAssetAlias))
		m.Handle("/get-asset", jsonHandler(a.getAsset))
		m.Handle("/list-assets", jsonHandler(a.listAssets))

		m.Handle("/create-key", jsonHandler(a.pseudohsmCreateKey))
		m.Handle("/update-key-alias", jsonHandler(a.pseudohsmUpdateKeyAlias))
		m.Handle("/list-keys", jsonHandler(a.pseudohsmListKeys))
		m.Handle("/delete-key", jsonHandler(a.pseudohsmDeleteKey))
		m.Handle("/reset-key-password", jsonHandler(a.pseudohsmResetPassword))
		m.Handle("/check-key-password", jsonHandler(a.pseudohsmCheckPassword))
		m.Handle("/sign-message", jsonHandler(a.signMessage))

		m.Handle("/build-transaction", jsonHandler(a.build))
		m.Handle("/build-chain-transactions", jsonHandler(a.buildChainTxs))
		m.Handle("/sign-transaction", jsonHandler(a.signTemplate))
		m.Handle("/sign-transactions", jsonHandler(a.signTemplates))

		m.Handle("/get-transaction", jsonHandler(a.getTransaction))
		m.Handle("/list-transactions", jsonHandler(a.listTransactions))

		m.Handle("/list-balances", jsonHandler(a.listBalances))
		m.Handle("/list-unspent-outputs", jsonHandler(a.listUnspentOutputs))

		m.Handle("/decode-program", jsonHandler(a.decodeProgram))

		m.Handle("/backup-wallet", jsonHandler(a.backupWalletImage))
		m.Handle("/restore-wallet", jsonHandler(a.restoreWalletImage))
		m.Handle("/rescan-wallet", jsonHandler(a.rescanWallet))
		m.Handle("/wallet-info", jsonHandler(a.getWalletInfo))
		m.Handle("/recovery-wallet", jsonHandler(a.recoveryFromRootXPubs))
	} else {
		log.Warn("Please enable wallet")
	}

	m.Handle("/", alwaysError(errors.New("not Found")))
	m.Handle("/error", jsonHandler(a.walletError))

	m.Handle("/create-access-token", jsonHandler(a.createAccessToken))
	m.Handle("/list-access-tokens", jsonHandler(a.listAccessTokens))
	m.Handle("/delete-access-token", jsonHandler(a.deleteAccessToken))
	m.Handle("/check-access-token", jsonHandler(a.checkAccessToken))

	m.Handle("/create-transaction-feed", jsonHandler(a.createTxFeed))
	m.Handle("/get-transaction-feed", jsonHandler(a.getTxFeed))
	m.Handle("/update-transaction-feed", jsonHandler(a.updateTxFeed))
	m.Handle("/delete-transaction-feed", jsonHandler(a.deleteTxFeed))
	m.Handle("/list-transaction-feeds", jsonHandler(a.listTxFeeds))

	m.Handle("/submit-transaction", jsonHandler(a.submit))
	m.Handle("/submit-transactions", jsonHandler(a.submitTxs))
	m.Handle("/estimate-transaction-gas", jsonHandler(a.estimateTxGas))

	m.Handle("/get-unconfirmed-transaction", jsonHandler(a.getUnconfirmedTx))
	m.Handle("/list-unconfirmed-transactions", jsonHandler(a.listUnconfirmedTxs))
	m.Handle("/decode-raw-transaction", jsonHandler(a.decodeRawTransaction))

	m.Handle("/get-block", jsonHandler(a.getBlock))
	m.Handle("/get-raw-block", jsonHandler(a.getRawBlock))
	m.Handle("/get-block-hash", jsonHandler(a.getBestBlockHash))
	m.Handle("/get-block-header", jsonHandler(a.getBlockHeader))
	m.Handle("/get-block-count", jsonHandler(a.getBlockCount))
	m.Handle("/get-difficulty", jsonHandler(a.getDifficulty))
	m.Handle("/get-hash-rate", jsonHandler(a.getHashRate))

	m.Handle("/is-mining", jsonHandler(a.isMining))
	m.Handle("/set-mining", jsonHandler(a.setMining))

	m.Handle("/get-work", jsonHandler(a.getWork))
	m.Handle("/get-work-json", jsonHandler(a.getWorkJSON))
	m.Handle("/submit-block", jsonHandler(a.submitBlock))
	m.Handle("/submit-work", jsonHandler(a.submitWork))
	m.Handle("/submit-work-json", jsonHandler(a.submitWorkJSON))

	m.Handle("/verify-message", jsonHandler(a.verifyMessage))
	m.Handle("/compile", jsonHandler(a.compileEquity))

	m.Handle("/gas-rate", jsonHandler(a.gasRate))
	m.Handle("/net-info", jsonHandler(a.getNetInfo))

	m.Handle("/list-peers", jsonHandler(a.listPeers))
	m.Handle("/disconnect-peer", jsonHandler(a.disconnectPeer))
	m.Handle("/connect-peer", jsonHandler(a.connectPeer))

	m.Handle("/get-merkle-proof", jsonHandler(a.getMerkleProof))

	m.HandleFunc("/websocket-subscribe", a.websocketHandler)

	handler := latencyHandler(m, walletEnable)
	handler = webAssetsHandler(handler)
	handler = gzip.Handler{Handler: handler}

	a.handler = handler
}

// json Handler
func jsonHandler(f interface{}) http.Handler {
	h, err := httpjson.Handler(f, errorFormatter.Write)
	if err != nil {
		panic(err)
	}
	return h
}

// error Handler
func alwaysError(err error) http.Handler {
	return jsonHandler(func() error { return err })
}

func webAssetsHandler(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", static.Handler{
		Assets:  dashboard.Files,
		Default: "index.html",
	}))
	mux.Handle("/equity/", http.StripPrefix("/equity/", static.Handler{
		Assets:  equity.Files,
		Default: "index.html",
	}))
	mux.Handle("/", next)

	return mux
}

// AuthHandler access token auth Handler
func AuthHandler(handler http.Handler, accessTokens *accesstoken.CredentialStore, authDisable bool) http.Handler {
	authenticator := authn.NewAPI(accessTokens, authDisable)

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// TODO(tessr): check that this path exists; return early if this path isn't legit
		req, err := authenticator.Authenticate(req)
		if err != nil {
			log.WithField("error", errors.Wrap(err, "Serve")).Error("Authenticate fail")
			err = errors.WithDetail(errNotAuthenticated, err.Error())
			errorFormatter.Write(req.Context(), rw, err)
			return
		}
		handler.ServeHTTP(rw, req)
	})
}

// RedirectHandler redirect to dashboard handler
func RedirectHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			http.Redirect(w, req, "/dashboard/", http.StatusFound)
			return
		}
		next.ServeHTTP(w, req)
	})
}

// latencyHandler take latency for the request url path, and redirect url path to wait-disable when wallet is closed
func latencyHandler(m *http.ServeMux, walletEnable bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// latency for the request url path
		if l := latency(m, req); l != nil {
			defer l.RecordSince(time.Now())
		}

		// when the wallet is not been opened and the url path is not been found, modify url path to error,
		// and redirect handler to error
		if _, pattern := m.Handler(req); pattern != req.URL.Path && !walletEnable {
			req.URL.Path = "/error"
			walletRedirectHandler(w, req)
			return
		}

		m.ServeHTTP(w, req)
	})
}

// walletRedirectHandler redirect to error when the wallet is closed
func walletRedirectHandler(w http.ResponseWriter, req *http.Request) {
	h := http.RedirectHandler(req.URL.String(), http.StatusMovedPermanently)
	h.ServeHTTP(w, req)
}
