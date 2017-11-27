package main

import (
	"bytes"
	"context"
	"encoding/hex"
	stdjson "encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bytom/blockchain"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/rpc"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/cmd/bytomcli/example"
	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/encoding/json"
	"github.com/bytom/env"
	"github.com/bytom/errors"
	"github.com/bytom/config"
)

// config vars
var (
	home    = blockchain.HomeDirFromEnvironment()
	coreURL = env.String("BYTOM_URL", "http://localhost:1999")

	// build vars; initialized by the linker
	buildTag    = "?"
	buildCommit = "?"
	buildDate   = "?"
)

type command struct {
	f func(*rpc.Client, []string)
}

type grantReq struct {
	Policy    string      `json:"policy"`
	GuardType string      `json:"guard_type"`
	GuardData interface{} `json:"guard_data"`
}

var commands = map[string]*command{
	"create-block-keypair":     {createBlockKeyPair},
	"reset":                    {reset},
	"grant":                    {grant},
	"revoke":                   {revoke},
	"wait":                     {wait},
	"create-account":           {createAccount},
	"bind-account":             {bindAccount},
	"update-account-tags":      {updateAccountTags},
	"create-asset":             {createAsset},
	"bind-asset":               {bindAsset},
	"update-asset-tags":        {updateAssetTags},
	"build-transaction":        {buildTransaction},
	"create-control-program":   {createControlProgram},
	"create-account-receiver":  {createAccountReceiver},
	"create-transaction-feed":  {createTxFeed},
	"get-transaction-feed":     {getTxFeed},
	"update-transaction-feed":  {updateTxFeed},
	"list-accounts":            {listAccounts},
	"list-assets":              {listAssets},
	"list-transaction-feeds":   {listTxFeeds},
	"list-transactions":        {listTransactions},
	"list-balances":            {listBalances},
	"list-unspent-outputs":     {listUnspentOutputs},
	"delete-transaction-feed":  {deleteTxFeed},
	"issue-test":               {example.IssueTest},
	"spend-test":               {example.SpendTest},
	"spend-coinbase-test":      {example.CoinbaseTest},
	"wallet-test":              {example.WalletTest},
	"create-access-token":      {createAccessToken},
	"list-access-token":        {listAccessTokens},
	"delete-access-token":      {deleteAccessToken},
	"check-access-token":       {checkAccessToken},
	"create-key":               {createKey},
	"list-keys":                {listKeys},
	"delete-key":               {deleteKey},
	"sign-transactions":        {signTransactions},
	"sub-create-issue-tx":      {submitCreateIssueTransaction},
	"sub-spend-account-tx":     {submitSpendTransaction},
	"reset-password":           {resetPassword},
	"update-alias":             {updateAlias},
	"net-info":                 {netInfo},
	"get-best-block-hash":      {getBestBlockHash},
	"get-block-header-by-hash": {getBlockHeaderByHash},
	"get-block-by-hash":        {getBlockByHash},
}

func main() {
	env.Parse()

	if len(os.Args) >= 2 && os.Args[1] == "-version" {
		var version string
		if buildTag != "?" {
			// build tag with bytom- prefix indicates official release
			version = strings.TrimPrefix(buildTag, "bytom-")
		} else {
			// version of the form rev123 indicates non-release build
			//version = rev.ID
		}
		fmt.Printf("bytomcli %s\n", version)
		return
	}

	if len(os.Args) < 2 {
		help(os.Stdout)
		os.Exit(0)
	}
	cmd := commands[os.Args[1]]
	if cmd == nil {
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		help(os.Stderr)
		os.Exit(1)
	}
	cmd.f(mustRPCClient(), os.Args[2:])
}

func createBlockKeyPair(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error: create-block-keypair takes no args")
	}
	pub := struct {
		Pub ed25519.PublicKey
	}{}
	err := client.Call(context.Background(), "/mockhsm/create-block-key", nil, &pub)
	dieOnRPCError(err)
	fmt.Printf("%x\n", pub.Pub)
}

// reset will attempt a reset rpc call on a remote core. If the
// core is not configured with reset capabilities an error is returned.
func reset(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error: reset takes no args")
	}

	req := map[string]bool{
		"Everything": true,
	}

	err := client.Call(context.Background(), "/reset", req, nil)
	dieOnRPCError(err)
}

func grant(client *rpc.Client, args []string) {
	editAuthz(client, args, "grant")
}

func revoke(client *rpc.Client, args []string) {
	editAuthz(client, args, "revoke")
}

func editAuthz(client *rpc.Client, args []string, action string) {
	usage := "usage: corectl " + action + " [policy] [guard]"
	var flags flag.FlagSet

	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
		fmt.Fprintln(os.Stderr, `
Where guard is one of:
  token=[id]   to affect an access token
  CN=[name]    to affect an X.509 Common Name
  OU=[name]    to affect an X.509 Organizational Unit

The type of guard (before the = sign) is case-insensitive.
`)
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()
	if len(args) != 2 {
		fatalln(usage)
	}

	req := grantReq{Policy: args[0]}

	switch typ, data := splitAfter2(args[1], "="); strings.ToUpper(typ) {
	case "TOKEN=":
		req.GuardType = "access_token"
		req.GuardData = map[string]interface{}{"id": data}
	case "CN=":
		req.GuardType = "x509"
		req.GuardData = map[string]interface{}{"subject": map[string]string{"CN": data}}
	case "OU=":
		req.GuardType = "x509"
		req.GuardData = map[string]interface{}{"subject": map[string]string{"OU": data}}
	default:
		fmt.Fprintln(os.Stderr, "unknown guard type", typ)
		fatalln(usage)
	}

	path := map[string]string{
		"grant":  "/create-authorization-grant",
		"revoke": "/delete-authorization-grant",
	}[action]
	err := client.Call(context.Background(), path, req, nil)
	dieOnRPCError(err)
}

func mustRPCClient() *rpc.Client {
	// TODO(kr): refactor some of this cert-loading logic into bytom/blockchain
	// and use it from cored as well.
	// Note that this function, unlike maybeUseTLS in cored,
	// does not load the cert and key from env vars,
	// only from the filesystem.
	certFile := filepath.Join(home, "tls.crt")
	keyFile := filepath.Join(home, "tls.key")
	config, err := blockchain.TLSConfig(certFile, keyFile, "")
	if err == blockchain.ErrNoTLS {
		return &rpc.Client{BaseURL: *coreURL}
	} else if err != nil {
		fatalln("error: loading TLS cert:", err)
	}

	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSClientConfig:       config,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	url := *coreURL
	if strings.HasPrefix(url, "http:") {
		url = "https:" + url[5:]
	}

	return &rpc.Client{
		BaseURL: url,
		Client:  &http.Client{Transport: t},
	}
}

func fatalln(v ...interface{}) {
	fmt.Printf("%v", v)
	os.Exit(2)
}

func dieOnRPCError(err error, prefixes ...interface{}) {
	if err == nil {
		return
	}

	if len(prefixes) > 0 {
		fmt.Fprintln(os.Stderr, prefixes...)
	}

	if msgErr, ok := errors.Root(err).(rpc.ErrStatusCode); ok && msgErr.ErrorData != nil {
		fmt.Fprintln(os.Stderr, "RPC error:", msgErr.ErrorData.ChainCode, msgErr.ErrorData.Message)
		if msgErr.ErrorData.Detail != "" {
			fmt.Fprintln(os.Stderr, "Detail:", msgErr.ErrorData.Detail)
		}
	} else {
		fmt.Fprintln(os.Stderr, "RPC error:", err)
	}

	os.Exit(2)
}

func help(w io.Writer) {
	fmt.Fprintln(w, "usage: corectl [-version] [command] [arguments]")
	fmt.Fprint(w, "\nThe commands are:\n\n")
	for name := range commands {
		fmt.Fprintln(w, "\t", name)
	}
	fmt.Fprint(w, "\nFlags:\n")
	fmt.Fprintln(w, "\t-version   print version information")
	fmt.Fprintln(w)
}

// splitAfter2 is like strings.SplitAfterN with n=2.
// If sep is not in s, it returns a="" and b=s.
func splitAfter2(s, sep string) (a, b string) {
	i := strings.Index(s, sep)
	k := i + len(sep)
	return s[:k], s[k:]
}

func wait(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error: wait takes no args")
	}

	for {
		err := client.Call(context.Background(), "/info", nil, nil)
		if err == nil {
			break
		}

		if statusErr, ok := errors.Root(err).(rpc.ErrStatusCode); ok && statusErr.StatusCode/100 != 5 {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func createAccount(client *rpc.Client, args []string) {
	if len(args) != 1 {
		fatalln("error: createAccount takes no args")
	}
	xprv, err := chainkd.NewXPrv(nil)
	if err != nil {
		fatalln("NewXprv error.")
	}
	xpub := xprv.XPub()
	fmt.Printf("xprv:%v\n", xprv)
	fmt.Printf("xpub:%v\n", xpub)
	type Ins struct {
		RootXPubs   []chainkd.XPub `json:"root_xpubs"`
		Quorum      int
		Alias       string
		Tags        map[string]interface{}
		ClientToken string `json:"client_token"`
	}
	var ins Ins
	ins.RootXPubs = []chainkd.XPub{xpub}
	ins.Quorum = 1
	ins.Alias = args[0]
	ins.Tags = map[string]interface{}{"test_tag": "v0"}
	ins.ClientToken = args[0]
	account := make([]query.AnnotatedAccount, 1)
	client.Call(context.Background(), "/create-account", &[]Ins{ins}, &account)
	fmt.Printf("responses:%v\n", account[0])
	fmt.Printf("account id:%v\n", account[0].ID)
}

func bindAccount(client *rpc.Client, args []string) {
	if len(args) != 2 {
		fatalln("error: bindAccount need args [account alias name] [account pub key]")
	}
	var xpub chainkd.XPub
	err := xpub.UnmarshalText([]byte(args[1]))
	if err == nil {
		fmt.Printf("xpub:%v\n", xpub)
	} else {
		fmt.Printf("xpub unmarshal error:%v\n", xpub)
	}
	type Ins struct {
		RootXPubs   []chainkd.XPub `json:"root_xpubs"`
		Quorum      int
		Alias       string
		Tags        map[string]interface{}
		ClientToken string `json:"client_token"`
	}
	var ins Ins
	ins.RootXPubs = []chainkd.XPub{xpub}
	ins.Quorum = 1
	ins.Alias = args[0]
	ins.Tags = map[string]interface{}{"test_tag": "v0"}
	ins.ClientToken = args[0]
	account := make([]query.AnnotatedAccount, 1)
	client.Call(context.Background(), "/create-account", &[]Ins{ins}, &account)
	fmt.Printf("responses:%v\n", account[0])
	fmt.Printf("account id:%v\n", account[0].ID)
}

func createAsset(client *rpc.Client, args []string) {
	if len(args) != 1 {
		fatalln("error: createAsset takes no args")
	}
	xprv, err := chainkd.NewXPrv(nil)
	if err != nil {
		fatalln("NewXprv error.")
	}
	xprv_, _ := xprv.MarshalText()
	xpub := xprv.XPub()
	fmt.Printf("xprv:%v\n", string(xprv_))
	xpub_, _ := xpub.MarshalText()
	fmt.Printf("xpub:%v\n", xpub_)
	type Ins struct {
		RootXPubs   []chainkd.XPub `json:"root_xpubs"`
		Quorum      int
		Alias       string
		Tags        map[string]interface{}
		Definition  map[string]interface{}
		ClientToken string `json:"client_token"`
	}
	var ins Ins
	ins.RootXPubs = []chainkd.XPub{xpub}
	ins.Quorum = 1
	ins.Alias = args[0]
	ins.Tags = map[string]interface{}{"test_tag": "v0"}
	ins.Definition = map[string]interface{}{}
	ins.ClientToken = args[0]
	assets := make([]query.AnnotatedAsset, 1)
	client.Call(context.Background(), "/create-asset", &[]Ins{ins}, &assets)
	fmt.Printf("responses:%v\n", assets)
	fmt.Printf("asset id:%v\n", assets[0].ID.String())
}

func bindAsset(client *rpc.Client, args []string) {
	if len(args) != 2 {
		fatalln("error: bindAsset need args [asset name] [asset xpub]")
	}
	var xpub chainkd.XPub
	err := xpub.UnmarshalText([]byte(args[1]))
	if err == nil {
		fmt.Printf("xpub:%v\n", xpub)
	} else {
		fmt.Printf("xpub unmarshal error:%v\n", xpub)
	}
	type Ins struct {
		RootXPubs   []chainkd.XPub `json:"root_xpubs"`
		Quorum      int
		Alias       string
		Tags        map[string]interface{}
		Definition  map[string]interface{}
		ClientToken string `json:"client_token"`
	}
	var ins Ins
	ins.RootXPubs = []chainkd.XPub{xpub}
	ins.Quorum = 1
	ins.Alias = args[0]
	ins.Tags = map[string]interface{}{"test_tag": "v0"}
	ins.Definition = map[string]interface{}{}
	ins.ClientToken = args[0]
	assets := make([]query.AnnotatedAsset, 1)
	client.Call(context.Background(), "/create-asset", &[]Ins{ins}, &assets)
	//dieOnRPCError(err)
	fmt.Printf("responses:%v\n", assets)
	fmt.Printf("asset id:%v\n", assets[0].ID.String())
}

func updateAccountTags(client *rpc.Client, args []string) {
	if len(args) != 2 {
		fatalln("update-account-tags [<ID>|<alias>] [tags_key:<tags_value>]")
	}

	type Ins struct {
		ID    *string
		Alias *string
		Tags  map[string]interface{} `json:"tags"`
	}
	var ins Ins

	//TODO:(1)when alias = acc...,how to do;
	//TODO:(2)support more tags together
	if "acc" == args[0][:3] {
		ins.ID = &args[0]
		ins.Alias = nil
	} else {
		ins.Alias = &args[0]
		ins.ID = nil
	}

	tags := strings.Split(args[1], ":")
	if len(tags) != 2 {
		fatalln("update-account-tags [<ID>|<alias>] [tags_key:<tags_value>]")
	}

	ins.Tags = map[string]interface{}{tags[0]: tags[1]}
	responses := make([]interface{}, 50)
	client.Call(context.Background(), "/update-account-tags", &[]Ins{ins}, &responses)
	fmt.Printf("responses:%v\n", responses)
}

func updateAssetTags(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error:updateAccountTags not use args")
	}
	type Ins struct {
		ID    *string
		Alias *string
		Tags  map[string]interface{} `json:"tags"`
	}
	var ins Ins
	id := "123456"
	alias := "asdfg"
	ins.ID = &id
	ins.Alias = &alias
	ins.Tags = map[string]interface{}{"test_tag": "v0"}
	responses := make([]interface{}, 50)
	client.Call(context.Background(), "/update-asset-tags", &[]Ins{ins}, &responses)
	fmt.Printf("responses:%v\n", responses)
}

func buildTransaction(client *rpc.Client, args []string) {
	if len(args) != 3 {
		fatalln("error: need args: [account id] [asset id] [file name]")
	}
	// Build Transaction.
	fmt.Printf("To build transaction:\n")
	// Now Issue actions
	buildReqFmt := `
		{"actions": [
			{
				"type":"spend_account_unspent_output",
				"receiver":null,
				"output_id":"%v",
				"reference_data":{}
			},
			{"type": "issue", "asset_id": "%s", "amount": 100},
			{"type": "control_account", "asset_id": "%s", "amount": 100, "account_id": "%s"}
		]}`
	buildReqStr := fmt.Sprintf(buildReqFmt, config.GenerateGenesisTx().ResultIds[0], args[1], args[1], args[0])
	var buildReq blockchain.BuildRequest
	err := stdjson.Unmarshal([]byte(buildReqStr), &buildReq)
	if err != nil {
		fmt.Printf("json Unmarshal error.")
	}

	tpl := make([]txbuilder.Template, 1)
	client.Call(context.Background(), "/build-transaction", []*blockchain.BuildRequest{&buildReq}, &tpl)
	marshalTpl, _ := stdjson.Marshal(tpl[0])
	fmt.Printf("tpl:%v\n", string(marshalTpl))
	file, _ := os.Create(args[2])
	defer file.Close()
	file.Write(marshalTpl)
}

func submitCreateIssueTransaction(client *rpc.Client, args []string) {
	if len(args) != 5 {
		fatalln("error: need args: [account1 id] [account2 id] [asset id] [asset xprv] [issue amount]")
	}
	// Build Transaction.
	fmt.Printf("To build transaction:\n")
	// Now Issue actions
	buildReqFmt := `
		{"actions": [
			{
				"type":"spend_account_unspent_output",
				"receiver":null,
				"output_id":"%v",
				"reference_data":{}
			},
			{"type": "issue", "asset_id": "%s", "amount": %s},
			{"type": "control_account", "asset_id": "%s", "amount": %s, "account_id": "%s"},
			{"type": "control_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": 8888888888, "account_id": "%s"},
			{"type": "control_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": 8888888888, "account_id": "%s"}
		]}`
	buildReqStr := fmt.Sprintf(buildReqFmt, config.GenerateGenesisTx().ResultIds[0], args[2], args[4], args[2], args[4], args[0], args[0], args[1])
	var buildReq blockchain.BuildRequest
	err := stdjson.Unmarshal([]byte(buildReqStr), &buildReq)
	if err != nil {
		fmt.Printf("json Unmarshal error.")
	}

	tpl := make([]txbuilder.Template, 1)
	client.Call(context.Background(), "/build-transaction", []*blockchain.BuildRequest{&buildReq}, &tpl)
	fmt.Printf("-----------tpl:%v\n", tpl[0])
	fmt.Printf("----------tpl transaction:%v\n", tpl[0].Transaction)
	fmt.Printf("----------btm inputs:%v\n", tpl[0].Transaction.Inputs[0])
	fmt.Printf("----------issue inputs:%v\n", tpl[0].Transaction.Inputs[1])

	var xprv_asset chainkd.XPrv
	fmt.Printf("xprv_asset:%v\n", args[3])
	xprv_asset.UnmarshalText([]byte(args[3]))
	// sign-transaction
	err = txbuilder.Sign(context.Background(), &tpl[0], []chainkd.XPub{xprv_asset.XPub()}, "", func(_ context.Context, _ chainkd.XPub, path [][]byte, data [32]byte, _ string) ([]byte, error) {
		derived := xprv_asset.Derive(path)
		return derived.Sign(data[:]), nil
	})
	if err != nil {
		fmt.Printf("sign-transaction error. err:%v\n", err)
	}
	fmt.Printf("sign tpl:%v\n", tpl[0])
	fmt.Printf("sign tpl's SigningInstructions:%v\n", tpl[0].SigningInstructions[0])
	fmt.Printf("SigningInstructions's SignatureWitnesses:%v\n", tpl[0].SigningInstructions[0].SignatureWitnesses[0])

	// submit-transaction
	var submitResponse interface{}
	submitArg := blockchain.SubmitArg{tpl, json.Duration{time.Duration(1000000)}, "none"}
	client.Call(context.Background(), "/submit-transaction", submitArg, &submitResponse)
	fmt.Printf("submit transaction:%v\n", submitResponse)
}

func submitSpendTransaction(client *rpc.Client, args []string) {
	if len(args) != 5 {
		fatalln("error: need args: [account1 id] [account2 id] [asset id] [account1 xprv] [spend amount]")
	}

	var xprvAccount1 chainkd.XPrv

	err := xprvAccount1.UnmarshalText([]byte(args[3]))
	if err == nil {
		fmt.Printf("xprv:%v\n", xprvAccount1)
	} else {
		fmt.Printf("xprv unmarshal error:%v\n", xprvAccount1)
		os.Exit(1)
	}
	// Build Transaction-Spend_account
	fmt.Printf("To build transaction:\n")
	buildReqFmt := `
		{"actions": [
		    {"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":20000000, "account_id": "%s"},
			{"type": "spend_account", "asset_id": "%s", "amount": %s, "account_id": "%s"},
			{"type": "control_account", "asset_id": "%s", "amount": %s, "account_id": "%s"}
	]}`

	buildReqStr := fmt.Sprintf(buildReqFmt, args[0], args[2], args[4], args[0], args[2], args[4], args[1])

	var buildReq blockchain.BuildRequest
	err = stdjson.Unmarshal([]byte(buildReqStr), &buildReq)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	tpl := make([]txbuilder.Template, 1)
	client.Call(context.Background(), "/build-transaction", []*blockchain.BuildRequest{&buildReq}, &tpl)
	fmt.Printf("tpl:%v\n", tpl)

	// sign-transaction-Spend_account
	err = txbuilder.Sign(context.Background(), &tpl[0], []chainkd.XPub{xprvAccount1.XPub()}, "", func(_ context.Context, _ chainkd.XPub, path [][]byte, data [32]byte, _ string) ([]byte, error) {
		derived := xprvAccount1.Derive(path)
		return derived.Sign(data[:]), nil
	})
	if err != nil {
		fmt.Printf("sign-transaction error. err:%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("sign tpl:%v\n", tpl[0])
	//fmt.Printf("sign tpl's SigningInstructions:%v\n", tpl[0].SigningInstructions[0])
	//fmt.Printf("SigningInstructions's SignatureWitnesses:%v\n", tpl[0].SigningInstructions[0].SignatureWitnesses[0])

	// submit-transaction-Spend_account
	var submitResponse interface{}
	submitArg := blockchain.SubmitArg{Transactions: tpl, Wait: json.Duration{Duration: time.Duration(1000000)}, WaitUntil: "none"}
	client.Call(context.Background(), "/submit-transaction", submitArg, &submitResponse)
	fmt.Printf("submit transaction:%v\n", submitResponse)
}

func createControlProgram(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error:createControlProgram not use args")
	}
	type Ins struct {
		Type   string
		Params stdjson.RawMessage
	}
	var ins Ins
	//TODO:undefined arguments to ins
	responses := make([]interface{}, 50)
	client.Call(context.Background(), "/create-control-program", &[]Ins{ins}, &responses)
	fmt.Printf("responses:%v\n", responses)
}

func createAccountReceiver(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error:createAccountReceiver not use args")
	}
	type Ins struct {
		AccountID    string    `json:"account_id"`
		AccountAlias string    `json:"account_alias"`
		ExpiresAt    time.Time `json:"expires_at"`
	}
	var ins Ins
	//TODO:undefined argument to ExpiresAt
	ins.AccountID = "123456"
	ins.AccountAlias = "zxcvbn"
	responses := make([]interface{}, 50)
	client.Call(context.Background(), "/create-Account-Receiver", &[]Ins{ins}, &responses)
	fmt.Printf("responses:%v\n", responses)
}

func createTxFeed(client *rpc.Client, args []string) {
	if len(args) != 2 {
		fatalln("error:createTxFeed need arguments")
	}
	type In struct {
		Alias  string
		Filter string
	}
	var in In
	in.Alias = args[0]
	in.Filter = args[1]

	client.Call(context.Background(), "/create-transaction-feed", in, nil)
}

func getTxFeed(client *rpc.Client, args []string) {
	if len(args) != 1 {
		fatalln("error:getTxFeed use args alias")
	}
	type requestQuery struct {
		Filter       string        `json:"filter,omitempty"`
		FilterParams []interface{} `json:"filter_params,omitempty"`
		SumBy        []string      `json:"sum_by,omitempty"`
		PageSize     int           `json:"page_size"`
		AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
		Timeout      json.Duration `json:"timeout"`
		After        string        `json:"after"`
		StartTimeMS  uint64        `json:"start_time,omitempty"`
		EndTimeMS    uint64        `json:"end_time,omitempty"`
		TimestampMS  uint64        `json:"timestamp,omitempty"`
		Type         string        `json:"type"`
		Aliases      []string      `json:"aliases,omitempty"`
	}
	var in requestQuery
	in.Filter = args[0]
	responses := make([]interface{}, 0)
	client.Call(context.Background(), "/get-transaction-feed", in, &responses)
	if len(responses) > 0 {
		for i, item := range responses {
			fmt.Println(i, "-----", item)
		}
	}
}

func updateTxFeed(client *rpc.Client, args []string) {
	if len(args) != 2 {
		fatalln("error:createTxFeed need arguments")
	}
	type In struct {
		Alias  string
		Filter string
	}
	var in In
	in.Alias = args[0]
	in.Filter = args[1]
	client.Call(context.Background(), "/update-transaction-feed", in, nil)
}

func deleteTxFeed(client *rpc.Client, args []string) {
	if len(args) != 1 {
		fatalln("error:deleteTxFeed use args alias")
	}
	type In struct {
		Alias string `json:"alias,omitempty"`
	}
	var in In
	in.Alias = args[0]
	client.Call(context.Background(), "/delete-transaction-feed", in, nil)
}

func listAccounts(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error:listAccounts not use args")
	}
	type requestQuery struct {
		Filter       string        `json:"filter,omitempty"`
		FilterParams []interface{} `json:"filter_params,omitempty"`
		SumBy        []string      `json:"sum_by,omitempty"`
		PageSize     int           `json:"page_size"`
		AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
		Timeout      json.Duration `json:"timeout"`
		After        string        `json:"after"`
		StartTimeMS  uint64        `json:"start_time,omitempty"`
		EndTimeMS    uint64        `json:"end_time,omitempty"`
		TimestampMS  uint64        `json:"timestamp,omitempty"`
		Type         string        `json:"type"`
		Aliases      []string      `json:"aliases,omitempty"`
	}
	var in requestQuery

	responses := make([]interface{}, 0)

	client.Call(context.Background(), "/list-accounts", in, &responses)
	if len(responses) > 0 {
		for i, item := range responses {
			fmt.Println(i, "-----", item)
		}
	}
}

func listAssets(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error:listAssets not use args")
	}
	type requestQuery struct {
		Filter       string        `json:"filter,omitempty"`
		FilterParams []interface{} `json:"filter_params,omitempty"`
		SumBy        []string      `json:"sum_by,omitempty"`
		PageSize     int           `json:"page_size"`
		AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
		Timeout      json.Duration `json:"timeout"`
		After        string        `json:"after"`
		StartTimeMS  uint64        `json:"start_time,omitempty"`
		EndTimeMS    uint64        `json:"end_time,omitempty"`
		TimestampMS  uint64        `json:"timestamp,omitempty"`
		Type         string        `json:"type"`
		Aliases      []string      `json:"aliases,omitempty"`
	}
	var in requestQuery
	responses := make([]interface{}, 0)

	client.Call(context.Background(), "/list-assets", in, &responses)
	if len(responses) > 0 {
		for i, item := range responses {
			fmt.Println(i, "-----", item)
		}
	}
}

func listTxFeeds(client *rpc.Client, args []string) {
	fmt.Println("listTxFeeds")
	if len(args) != 0 {
		fatalln("error:listTxFeeds not use args")
	}
	type requestQuery struct {
		Filter       string        `json:"filter,omitempty"`
		FilterParams []interface{} `json:"filter_params,omitempty"`
		SumBy        []string      `json:"sum_by,omitempty"`
		PageSize     int           `json:"page_size"`
		AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
		Timeout      json.Duration `json:"timeout"`
		After        string        `json:"after"`
		StartTimeMS  uint64        `json:"start_time,omitempty"`
		EndTimeMS    uint64        `json:"end_time,omitempty"`
		TimestampMS  uint64        `json:"timestamp,omitempty"`
		Type         string        `json:"type"`
		Aliases      []string      `json:"aliases,omitempty"`
	}
	var in requestQuery

	responses := make([]interface{}, 0)

	client.Call(context.Background(), "/list-transaction-feeds", in, &responses)
	if len(responses) > 0 {
		for i, item := range responses {
			fmt.Println(i, "-----", item)
		}
	}

}

func listTransactions(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error:listTransactions not use args")
	}
	type requestQuery struct {
		Filter       string        `json:"filter,omitempty"`
		FilterParams []interface{} `json:"filter_params,omitempty"`
		SumBy        []string      `json:"sum_by,omitempty"`
		PageSize     int           `json:"page_size"`
		AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
		Timeout      json.Duration `json:"timeout"`
		After        string        `json:"after"`
		StartTimeMS  uint64        `json:"start_time,omitempty"`
		EndTimeMS    uint64        `json:"end_time,omitempty"`
		TimestampMS  uint64        `json:"timestamp,omitempty"`
		Type         string        `json:"type"`
		Aliases      []string      `json:"aliases,omitempty"`
	}
	var in requestQuery
	var rawResponse []byte
	var response blockchain.Response

	client.Call(context.Background(), "/list-transactions", in, &rawResponse)

	if err := stdjson.Unmarshal(rawResponse, &response); err != nil {
		fmt.Println(err)
	}

	if response.Status != blockchain.SUCCESS {
		fmt.Println(response.Msg)
		return
	}

	for i, item := range response.Data {
		fmt.Println(i, "-----", item)
	}

}

func listBalances(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error:listBalances not use args")
	}
	type requestQuery struct {
		Filter       string        `json:"filter,omitempty"`
		FilterParams []interface{} `json:"filter_params,omitempty"`
		SumBy        []string      `json:"sum_by,omitempty"`
		PageSize     int           `json:"page_size"`
		AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
		Timeout      json.Duration `json:"timeout"`
		After        string        `json:"after"`
		StartTimeMS  uint64        `json:"start_time,omitempty"`
		EndTimeMS    uint64        `json:"end_time,omitempty"`
		TimestampMS  uint64        `json:"timestamp,omitempty"`
		Type         string        `json:"type"`
		Aliases      []string      `json:"aliases,omitempty"`
	}

	var in requestQuery
	responses := make([]interface{}, 0)

	client.Call(context.Background(), "/list-balances", in, &responses)
	if len(responses) > 0 {
		for i, item := range responses {
			fmt.Println(i, "-----", item)
		}
	}
}

func listUnspentOutputs(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error:listUnspentOutputs not use args")
	}
	type requestQuery struct {
		Filter       string        `json:"filter,omitempty"`
		FilterParams []interface{} `json:"filter_params,omitempty"`
		SumBy        []string      `json:"sum_by,omitempty"`
		PageSize     int           `json:"page_size"`
		AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
		Timeout      json.Duration `json:"timeout"`
		After        string        `json:"after"`
		StartTimeMS  uint64        `json:"start_time,omitempty"`
		EndTimeMS    uint64        `json:"end_time,omitempty"`
		TimestampMS  uint64        `json:"timestamp,omitempty"`
		Type         string        `json:"type"`
		Aliases      []string      `json:"aliases,omitempty"`
	}
	var in requestQuery
	responses := make([]interface{}, 0)

	client.Call(context.Background(), "/list-unspent-outputs", in, &responses)
	if len(responses) > 0 {
		for i, item := range responses {
			fmt.Println(i, "-----", item)
		}
	}
}

func createAccessToken(client *rpc.Client, args []string) {
	if len(args) != 1 {
		fatalln("error:createAccessToken use args id")
	}
	type Token struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	var token Token
	token.ID = args[0]

	var response interface{}

	client.Call(context.Background(), "/create-access-token", &token, &response)
	fmt.Println(response)
}

func listAccessTokens(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error:listAccessTokens not use args")
	}
	var response interface{}
	client.Call(context.Background(), "/list-access-token", nil, &response)
	fmt.Println(response)
}

func deleteAccessToken(client *rpc.Client, args []string) {
	if len(args) != 1 {
		fatalln("error:deleteAccessToken use args id")
	}
	type Token struct {
		ID     string `json:"id"`
		Secert string `json:"secert,omitempty"`
	}
	var token Token
	token.ID = args[0]
	var response interface{}
	client.Call(context.Background(), "/delete-access-token", &token, &response)
	fmt.Println(response)
}

func checkAccessToken(client *rpc.Client, args []string) {
	if len(args) != 1 {
		fatalln("error:deleteAccessToken use args token")
	}
	type Token struct {
		ID     string `json:"id"`
		Secret string `json:"secret,omitempty"`
	}
	var token Token
	inputs := strings.Split(args[0], ":")
	token.ID = inputs[0]
	token.Secret = inputs[1]
	var response interface{}
	client.Call(context.Background(), "/check-access-token", &token, &response)
	fmt.Println(response)
}

func createKey(client *rpc.Client, args []string) {
	if len(args) != 2 {
		fatalln("error: createKey args not vaild")
	}
	type Key struct {
		Alias    string
		Password string
	}
	var key Key
	var response map[string]interface{}
	key.Alias = args[0]
	key.Password = args[1]

	client.Call(context.Background(), "/create-key", &key, &response)
	fmt.Printf("Alias: %v,  XPub: %v, File: %v\n", response["alias"], response["xpub"], response["file"])
}

func deleteKey(client *rpc.Client, args []string) {
	if len(args) != 2 {
		fatalln("error: deleteKey args not vaild")
	}
	type Key struct {
		Password string
		XPub     chainkd.XPub `json:"xpubs"`
	}
	var key Key
	xpub := new(chainkd.XPub)
	data, err := hex.DecodeString(args[1])
	if err != nil {
		fatalln("error: deletKey %v", err)
	}
	copy(xpub[:], data)
	key.Password = args[0]
	key.XPub = *xpub
	client.Call(context.Background(), "/delete-key", &key, nil)
}

func listKeys(client *rpc.Client, args []string) {
	if len(args) != 2 {
		fatalln("error: listKeys args not vaild")
	}

	type requestQuery struct {
		Filter       string        `json:"filter,omitempty"`
		FilterParams []interface{} `json:"filter_params,omitempty"`
		SumBy        []string      `json:"sum_by,omitempty"`
		PageSize     int           `json:"page_size"`
		AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
		Timeout      json.Duration `json:"timeout"`
		After        string        `json:"after"`
		StartTimeMS  uint64        `json:"start_time,omitempty"`
		EndTimeMS    uint64        `json:"end_time,omitempty"`
		TimestampMS  uint64        `json:"timestamp,omitempty"`
		Type         string        `json:"type"`
		Aliases      []string      `json:"aliases,omitempty"`
	}
	var in requestQuery
	in.After = args[0]
	in.PageSize, _ = strconv.Atoi(args[1])
	var response map[string][]interface{}
	client.Call(context.Background(), "/list-keys", &in, &response)
	for i, item := range response["items"] {
		key := item.(map[string]interface{})
		fmt.Printf("---No.%v Alias:%v Address:%v File:%v\n", i, key["alias"], key["address"], key["file"])
	}
}

func signTransactions(client *rpc.Client, args []string) {

	// sign-transaction
	type param struct {
		Auth  string
		Txs   []*txbuilder.Template `json:"transactions"`
		XPubs chainkd.XPub          `json:"xpubs"`
		XPrv  chainkd.XPrv          `json:"xprv"`
	}

	var in param
	var xprv chainkd.XPrv
	var xpub chainkd.XPub
	var err error

	if len(args) == 3 {
		err = xpub.UnmarshalText([]byte(args[1]))
		if err == nil {
			fmt.Printf("xpub:%v\n", xpub)
		} else {
			fmt.Printf("xpub unmarshal error:%v\n", xpub)
		}
		in.XPubs = xpub
		in.Auth = args[2]

	} else if len(args) == 2 {
		err = xprv.UnmarshalText([]byte(args[1]))
		if err == nil {
			fmt.Printf("xprv:%v\n", xprv)
		} else {
			fmt.Printf("xprv unmarshal error:%v\n", xprv)
		}
		in.XPrv = xprv

	} else {
		fatalln("error: signTransaction need args: [tpl file name] [xPub] [password], 3 args not equal"+
			"or [tpl file name] [xPrv], 2 args not equal ", len(args))
	}

	var tpl txbuilder.Template
	file, _ := os.Open(args[0])
	tpl_byte := make([]byte, 10000)
	file.Read(tpl_byte)
	fmt.Printf("tpl_byte:%v\n", string(tpl_byte))
	err = stdjson.Unmarshal(bytes.Trim(tpl_byte, "\x00"), &tpl)
	fmt.Printf("tpl:%v, err:%v\n", tpl, err)
	in.Txs = []*txbuilder.Template{&tpl}

	var response = make([]interface{}, 1)
	client.Call(context.Background(), "/sign-transactions", &in, &response)
	fmt.Printf("sign response:%v\n", response)
}

func resetPassword(client *rpc.Client, args []string) {
	if len(args) != 3 {
		fatalln("error: resetpassword args not vaild")
	}
	type Key struct {
		OldPassword string
		NewPassword string
		XPub        chainkd.XPub `json:"xpubs"`
	}
	var key Key
	xpub := new(chainkd.XPub)
	data, err := hex.DecodeString(args[2])
	if err != nil {
		fatalln("error: resetPassword %v", err)
	}
	copy(xpub[:], data)
	key.OldPassword = args[0]
	key.NewPassword = args[1]
	key.XPub = *xpub
	client.Call(context.Background(), "/reset-password", &key, nil)
}

func updateAlias(client *rpc.Client, args []string) {
	if len(args) != 3 {
		fatalln("error: resetpassword args not vaild")
	}
	type Key struct {
		Password string
		NewAlias string
		XPub     chainkd.XPub `json:"xpubs"`
	}
	var key Key
	xpub := new(chainkd.XPub)
	data, err := hex.DecodeString(args[2])
	if err != nil {
		fatalln("error: resetPassword %v", err)
	}
	copy(xpub[:], data)
	key.Password = args[0]
	key.NewAlias = args[1]
	key.XPub = *xpub
	client.Call(context.Background(), "/update-alias", &key, nil)
}

func netInfo(client *rpc.Client, args []string) {
	var response interface{}
	client.Call(context.Background(), "/net-info", nil, &response)
	fmt.Printf("net info:%v\n", response)
}

func getBestBlockHash(client *rpc.Client, args []string) {
	var response interface{}
	client.Call(context.Background(), "/get-best-block-hash", nil, &response)
	fmt.Printf("best-block-hash:%v\n", response)
}

func getBlockHeaderByHash(client *rpc.Client, args []string) {
	if len(args) != 1 {
		fatalln("error: get-block-header-by-hash args not valid: get-block-header-by-hash [hash]")
	}
	var response interface{}
	client.Call(context.Background(), "/get-block-header-by-hash", args[0], &response)
	fmt.Printf("block header: %v\n", response)
}

func getBlockByHash(client *rpc.Client, args []string) {
	if len(args) != 1 {
		fatalln("error: get-block-by-hash args not valid: get-block-by-hash [hash]")
	}
	var response interface{}
	client.Call(context.Background(), "/get-block-by-hash", args[0], &response)
	fmt.Printf("%v\n", response)
}
