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
	"github.com/bytom/dashboard"
	"github.com/bytom/errors"
	"github.com/bytom/mining/cpuminer"
	"github.com/bytom/mining/miningpool"
	"github.com/bytom/net/http/authn"
	"github.com/bytom/net/http/gzip"
	"github.com/bytom/net/http/httpjson"
	"github.com/bytom/net/http/static"
	"github.com/bytom/netsync"
	"github.com/bytom/protocol"
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
	FAIL               = "fail"
	crosscoreRPCPrefix = "/rpc/"
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
	sync          *netsync.SyncManager
	wallet        *wallet.Wallet
	accessTokens  *accesstoken.CredentialStore
	chain         *protocol.Chain
	server        *http.Server
	handler       http.Handler
	txFeedTracker *txfeed.Tracker
	cpuMiner      *cpuminer.CPUMiner
	miningPool    *miningpool.MiningPool
}

func (a *API) initServer(config *cfg.Config) {
	// The waitHandler accepts incoming requests, but blocks until its underlying
	// handler is set, when the second phase is complete.
	var coreHandler waitHandler
	var handler http.Handler

	coreHandler.wg.Add(1)
	mux := http.NewServeMux()
	mux.Handle("/", &coreHandler)

	handler = mux
	if config.Auth.Disable == false {
		handler = AuthHandler(handler, a.accessTokens)
	}
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
func NewAPI(sync *netsync.SyncManager, wallet *wallet.Wallet, txfeeds *txfeed.Tracker, cpuMiner *cpuminer.CPUMiner, miningPool *miningpool.MiningPool, chain *protocol.Chain, config *cfg.Config, token *accesstoken.CredentialStore) *API {
	api := &API{
		sync:          sync,
		wallet:        wallet,
		chain:         chain,
		accessTokens:  token,
		txFeedTracker: txfeeds,
		cpuMiner:      cpuMiner,
		miningPool:    miningPool,
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
		m.Handle("/update-account-tags", jsonHandler(a.updateAccountTags))
		m.Handle("/list-accounts", jsonHandler(a.listAccounts))
		m.Handle("/delete-account", jsonHandler(a.deleteAccount))

		m.Handle("/create-account-receiver", jsonHandler(a.createAccountReceiver))
		m.Handle("/list-addresses", jsonHandler(a.listAddresses))
		m.Handle("/validate-address", jsonHandler(a.validateAddress))

		m.Handle("/create-asset", jsonHandler(a.createAsset))
		m.Handle("/update-asset-alias", jsonHandler(a.updateAssetAlias))
		m.Handle("/update-asset-tags", jsonHandler(a.updateAssetTags))
		m.Handle("/list-assets", jsonHandler(a.listAssets))

		m.Handle("/create-key", jsonHandler(a.pseudohsmCreateKey))
		m.Handle("/list-keys", jsonHandler(a.pseudohsmListKeys))
		m.Handle("/delete-key", jsonHandler(a.pseudohsmDeleteKey))
		m.Handle("/reset-key-password", jsonHandler(a.pseudohsmResetPassword))

		m.Handle("/export-private-key", jsonHandler(a.walletExportKey))
		m.Handle("/import-private-key", jsonHandler(a.walletImportKey))
		m.Handle("/import-key-progress", jsonHandler(a.keyImportProgress))

		m.Handle("/build-transaction", jsonHandler(a.build))
		m.Handle("/sign-transaction", jsonHandler(a.pseudohsmSignTemplates))
		m.Handle("/submit-transaction", jsonHandler(a.submit))
		// TODO remove this api, separate sign and submit process
		m.Handle("/sign-submit-transaction", jsonHandler(a.signSubmit))
		m.Handle("/get-transaction", jsonHandler(a.getTransaction))
		m.Handle("/list-transactions", jsonHandler(a.listTransactions))

		m.Handle("/list-balances", jsonHandler(a.listBalances))
		m.Handle("/list-unspent-outputs", jsonHandler(a.listUnspentOutputs))
	} else {
		log.Warn("Please enable wallet")
	}

	m.Handle("/", alwaysError(errors.New("not Found")))
	m.Handle("/error", jsonHandler(a.walletError))

	m.Handle("/net-info", jsonHandler(a.getNetInfo))

	m.Handle("/create-access-token", jsonHandler(a.createAccessToken))
	m.Handle("/list-access-tokens", jsonHandler(a.listAccessTokens))
	m.Handle("/delete-access-token", jsonHandler(a.deleteAccessToken))
	m.Handle("/check-access-token", jsonHandler(a.checkAccessToken))

	m.Handle("/create-transaction-feed", jsonHandler(a.createTxFeed))
	m.Handle("/get-transaction-feed", jsonHandler(a.getTxFeed))
	m.Handle("/update-transaction-feed", jsonHandler(a.updateTxFeed))
	m.Handle("/delete-transaction-feed", jsonHandler(a.deleteTxFeed))
	m.Handle("/list-transaction-feeds", jsonHandler(a.listTxFeeds))

	m.Handle("/block-hash", jsonHandler(a.getBestBlockHash))
	m.Handle("/get-block-header-by-hash", jsonHandler(a.getBlockHeaderByHash))
	m.Handle("/get-block-header-by-height", jsonHandler(a.getBlockHeaderByHeight))
	m.Handle("/get-block", jsonHandler(a.getBlock))
	m.Handle("/get-block-count", jsonHandler(a.getBlockCount))
	m.Handle("/get-block-transactions-count-by-hash", jsonHandler(a.getBlockTransactionsCountByHash))
	m.Handle("/get-block-transactions-count-by-height", jsonHandler(a.getBlockTransactionsCountByHeight))

	m.Handle("/is-mining", jsonHandler(a.isMining))
	m.Handle("/gas-rate", jsonHandler(a.gasRate))
	m.Handle("/get-work", jsonHandler(a.getWork))
	m.Handle("/submit-work", jsonHandler(a.submitWork))
	m.Handle("/set-mining", jsonHandler(a.setMining))

	handler := latencyHandler(m, walletEnable)
	handler = maxBytesHandler(handler) // TODO(tessr): consider moving this to non-core specific mux
	handler = webAssetsHandler(handler)
	handler = gzip.Handler{Handler: handler}

	a.handler = handler
}

func maxBytesHandler(h http.Handler) http.Handler {
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
	mux.Handle("/", next)

	return mux
}

// AuthHandler access token auth Handler
func AuthHandler(handler http.Handler, accessTokens *accesstoken.CredentialStore) http.Handler {
	authenticator := authn.NewAPI(accessTokens)

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// TODO(tessr): check that this path exists; return early if this path isn't legit
		req, err := authenticator.Authenticate(req)
		if err != nil {
			log.WithField("error", errors.Wrap(err, "Serve")).Error("Authenticate fail")
			err = errors.Sub(errNotAuthenticated, err)
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
