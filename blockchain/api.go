package blockchain

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/kr/secureheader"
	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/bytom/blockchain/accesstoken"
	cfg "github.com/bytom/config"
	"github.com/bytom/dashboard"
	"github.com/bytom/errors"
	"github.com/bytom/net/http/authn"
	"github.com/bytom/net/http/httpjson"
	"github.com/bytom/net/http/static"
)

var (
	errNotAuthenticated = errors.New("not authenticated")
	httpReadTimeout     = 2 * time.Minute
	httpWriteTimeout    = time.Hour
)

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

type API struct {
	bcr     *BlockchainReactor
	server  *http.Server
	handler http.Handler
}

func (a *API) initServer(config *cfg.Config) {
	// The waitHandler accepts incoming requests, but blocks until its underlying
	// handler is set, when the second phase is complete.
	var coreHandler waitHandler
	coreHandler.wg.Add(1)
	mux := http.NewServeMux()
	mux.Handle("/", &coreHandler)

	var handler http.Handler = mux

	if config.Auth.Disable == false {
		handler = AuthHandler(handler, a.bcr.wallet.Tokens)
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

func NewAPI(bcr *BlockchainReactor, config *cfg.Config) *API {
	api := &API{
		bcr: bcr,
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
	m := http.NewServeMux()
	if a.bcr.wallet != nil && a.bcr.wallet.AccountMgr != nil && a.bcr.wallet.AssetReg != nil {
		m.Handle("/create-account", jsonHandler(a.bcr.createAccount))
		m.Handle("/update-account-tags", jsonHandler(a.bcr.updateAccountTags))
		m.Handle("/create-account-receiver", jsonHandler(a.bcr.createAccountReceiver))
		m.Handle("/list-accounts", jsonHandler(a.bcr.listAccounts))
		m.Handle("/list-addresses", jsonHandler(a.bcr.listAddresses))
		m.Handle("/delete-account", jsonHandler(a.bcr.deleteAccount))
		m.Handle("/validate-address", jsonHandler(a.bcr.validateAddress))

		m.Handle("/create-asset", jsonHandler(a.bcr.createAsset))
		m.Handle("/update-asset-alias", jsonHandler(a.bcr.updateAssetAlias))
		m.Handle("/update-asset-tags", jsonHandler(a.bcr.updateAssetTags))
		m.Handle("/list-assets", jsonHandler(a.bcr.listAssets))

		m.Handle("/create-key", jsonHandler(a.bcr.pseudohsmCreateKey))
		m.Handle("/list-keys", jsonHandler(a.bcr.pseudohsmListKeys))
		m.Handle("/delete-key", jsonHandler(a.bcr.pseudohsmDeleteKey))
		m.Handle("/reset-key-password", jsonHandler(a.bcr.pseudohsmResetPassword))

		m.Handle("/get-transaction", jsonHandler(a.bcr.getTransaction))
		m.Handle("/list-transactions", jsonHandler(a.bcr.listTransactions))
		m.Handle("/list-balances", jsonHandler(a.bcr.listBalances))
	} else {
		log.Warn("Please enable wallet")
	}

	m.Handle("/", alwaysError(errors.New("not Found")))

	m.Handle("/build-transaction", jsonHandler(a.bcr.build))
	m.Handle("/sign-transaction", jsonHandler(a.bcr.pseudohsmSignTemplates))
	m.Handle("/submit-transaction", jsonHandler(a.bcr.submit))
	m.Handle("/sign-submit-transaction", jsonHandler(a.bcr.signSubmit))

	m.Handle("/create-transaction-feed", jsonHandler(a.bcr.createTxFeed))
	m.Handle("/get-transaction-feed", jsonHandler(a.bcr.getTxFeed))
	m.Handle("/update-transaction-feed", jsonHandler(a.bcr.updateTxFeed))
	m.Handle("/delete-transaction-feed", jsonHandler(a.bcr.deleteTxFeed))
	m.Handle("/list-transaction-feeds", jsonHandler(a.bcr.listTxFeeds))
	m.Handle("/list-unspent-outputs", jsonHandler(a.bcr.listUnspentOutputs))
	m.Handle("/info", jsonHandler(a.bcr.info))

	m.Handle("/create-access-token", jsonHandler(a.bcr.createAccessToken))
	m.Handle("/list-access-tokens", jsonHandler(a.bcr.listAccessTokens))
	m.Handle("/delete-access-token", jsonHandler(a.bcr.deleteAccessToken))
	m.Handle("/check-access-token", jsonHandler(a.bcr.checkAccessToken))

	m.Handle("/block-hash", jsonHandler(a.bcr.getBestBlockHash))

	m.Handle("/export-private-key", jsonHandler(a.bcr.walletExportKey))
	m.Handle("/import-private-key", jsonHandler(a.bcr.walletImportKey))
	m.Handle("/import-key-progress", jsonHandler(a.bcr.keyImportProgress))

	m.Handle("/get-block-header-by-hash", jsonHandler(a.bcr.getBlockHeaderByHash))
	m.Handle("/get-block-header-by-height", jsonHandler(a.bcr.getBlockHeaderByHeight))
	m.Handle("/get-block", jsonHandler(a.bcr.getBlock))
	m.Handle("/get-block-count", jsonHandler(a.bcr.getBlockCount))
	m.Handle("/get-block-transactions-count-by-hash", jsonHandler(a.bcr.getBlockTransactionsCountByHash))
	m.Handle("/get-block-transactions-count-by-height", jsonHandler(a.bcr.getBlockTransactionsCountByHeight))

	m.Handle("/net-info", jsonHandler(a.bcr.getNetInfo))

	m.Handle("/is-mining", jsonHandler(a.bcr.isMining))
	m.Handle("/gas-rate", jsonHandler(a.bcr.gasRate))
	m.Handle("/getwork", jsonHandler(a.bcr.getWork))
	m.Handle("/submitwork", jsonHandler(a.bcr.submitWork))

	latencyHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if l := latency(m, req); l != nil {
			defer l.RecordSince(time.Now())
		}
		m.ServeHTTP(w, req)
	})
	handler := maxBytes(latencyHandler) // TODO(tessr): consider moving this to non-core specific mux
	handler = webAssetsHandler(handler)

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
	mux.Handle("/", next)

	return mux
}

//AuthHandler access token auth Handler
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

func RedirectHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			http.Redirect(w, req, "/dashboard/", http.StatusFound)
			return
		}
		next.ServeHTTP(w, req)
	})
}
