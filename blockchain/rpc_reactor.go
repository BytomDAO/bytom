package blockchain

import (
	"net/http"
	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
	"github.com/bytom/net/http/httpjson"
)

// json handler
func jsonHandler(f interface{}) http.Handler {
	h, err := httpjson.Handler(f, errorFormatter.Write)
	if err != nil {
		panic(err)
	}
	return h
}

// error handler
func alwaysError(err error) http.Handler {
	return jsonHandler(func() error { return err })
}

// serve http
func (bcr *BlockchainReactor) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	bcr.handler.ServeHTTP(rw, req)
}

// Build json rpc handler
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
}
