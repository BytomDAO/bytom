// Command corectl provides miscellaneous control functions for a Chain Core.
package main

import (
	"bytes"
	"context"
//	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	//"strconv"
	"strings"
	"time"
	stdjson "encoding/json"

	"github.com/bytom/blockchain"
//	"chain/core/accesstoken"
	//"github.com/bytom/config"
	"github.com/bytom/blockchain/rpc"
	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/env"
	"github.com/bytom/errors"
//	"github.com/bytom/generated/rev"
	"github.com/bytom/log"
	"github.com/bytom/crypto/ed25519/chainkd"
	//"github.com/bytom/protocol/bc"
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

// We collect log output in this buffer,
// and display it only when there's an error.
var logbuf bytes.Buffer

type command struct {
	f func(*rpc.Client, []string)
}

type grantReq struct {
	Policy    string      `json:"policy"`
	GuardType string      `json:"guard_type"`
	GuardData interface{} `json:"guard_data"`
}

var commands = map[string]*command{
	//"config-generator":     {configGenerator},
	"create-block-keypair": {createBlockKeyPair},
	//"create-token":         {createToken},
	//"config":               {configNongenerator},
	"reset":                {reset},
	"grant":                {grant},
	"revoke":               {revoke},
	"wait":                 {wait},
	"create-account":       {createAccount},
	"update-account-tags":  {updateAccountTags},
	"create-asset":		{createAsset},
	"update-asset-tags":	{updateAssetTags},
	"create-control-program": {createControlProgram},
	"create-account-receiver": {createAccountReceiver},
	"create-transaction-feed": {createTxFeed},
	"get-transaction-feed":    {getTxFeed},
	"update-transaction-feed": {updateTxFeed},
	"delete-transaction-feed": {deleteTxFeed},
}

func main() {
	log.SetOutput(&logbuf)
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
		fmt.Printf("build-commit: %v\n", buildCommit)
		fmt.Printf("build-date: %v\n", buildDate)
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

/*
func configGenerator(client *rpc.Client, args []string) {
	const usage = "usage: corectl config-generator [flags] [quorum] [pubkey url]..."
	var (
		quorum  uint32
		signers []*config.BlockSigner
		err     error
	)

	var flags flag.FlagSet
	maxIssuanceWindow := flags.Duration("w", 24*time.Hour, "the maximum issuance window `duration` for this generator")
	flagK := flags.String("k", "", "local `pubkey` for signing blocks")
	flagHSMURL := flags.String("hsm-url", "", "hsm `url` for signing blocks (mockhsm if empty)")
	flagHSMToken := flags.String("hsm-token", "", "hsm `access-token` for connecting to hsm")

	flags.Usage = func() {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()

	// not a blocksigner
	if *flagK == "" && *flagHSMURL != "" {
		fatalln("error: flag -hsm-url has no effect without -k")
	}

	// TODO(ameets): update when switching to x.509 authorization
	if (*flagHSMURL == "") != (*flagHSMToken == "") {
		fatalln("error: flags -hsm-url and -hsm-token must be given together")
	}

	if len(args) == 0 {
		if *flagK != "" {
			quorum = 1
		}
	} else if len(args)%2 != 1 {
		fatalln(usage)
	} else {
		q64, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			fatalln(usage)
		}
		quorum = uint32(q64)

		for i := 1; i < len(args); i += 2 {
			pubkey, err := hex.DecodeString(args[i])
			if err != nil {
				fatalln(usage)
			}
			if len(pubkey) != ed25519.PublicKeySize {
				fatalln("error:", "bad ed25519 public key length")
			}
			url := args[i+1]
			signers = append(signers, &config.BlockSigner{
				Pubkey: pubkey,
				Url:    url,
			})
		}
	}

	var blockPub []byte
	if *flagK != "" {
		blockPub, err = hex.DecodeString(*flagK)
		if err != nil {
			fatalln("error: unable to decode block pub")
		}
	}

	conf := &config.Config{
		IsGenerator:         true,
		Quorum:              quorum,
		Signers:             signers,
		MaxIssuanceWindowMs: bc.DurationMillis(*maxIssuanceWindow),
		IsSigner:            *flagK != "",
		BlockPub:            blockPub,
		BlockHsmUrl:         *flagHSMURL,
		BlockHsmAccessToken: *flagHSMToken,
	}

	err = client.Call(context.Background(), "/configure", conf, nil)
	dieOnRPCError(err)

	wait(client, nil)
	var r map[string]interface{}
	err = client.Call(context.Background(), "/info", nil, &r)
	dieOnRPCError(err)
	fmt.Println(r["blockchain_id"])
}
*/

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

/*
func createToken(client *rpc.Client, args []string) {
	const usage = "usage: corectl create-token [-net] [name] [policy]"
	var flags flag.FlagSet
	flagNet := flags.Bool("net", false, "DEPRECATED. create a network token instead of client")
	flags.Usage = func() {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()
	if len(args) == 2 && *flagNet || len(args) < 1 || len(args) > 2 {
		fatalln(usage)
	}

	req := struct{ ID string }{args[0]}
	var tok accesstoken.Token
	// TODO(kr): find a way to make this atomic with the grant below
	err := client.Call(context.Background(), "/create-access-token", req, &tok)
	dieOnRPCError(err)
	fmt.Println(tok.Token)

	grant := grantReq{
		GuardType: "access_token",
		GuardData: map[string]string{"id": tok.ID},
	}
	switch {
	case len(args) == 2:
		grant.Policy = args[1]
	case *flagNet:
		grant.Policy = "crosscore"
		fmt.Fprintln(os.Stderr, "warning: the network flag is deprecated")
	default:
		grant.Policy = "client-readwrite"
		fmt.Fprintln(os.Stderr, "warning: implicit policy name is deprecated")
	}
	err = client.Call(context.Background(), "/create-authorization-grant", grant, nil)
	dieOnRPCError(err, "Auth grant error:")
}
*/

/*
func configNongenerator(client *rpc.Client, args []string) {
	const usage = "usage: corectl config [flags] [blockchain-id] [generator-url]"
	var flags flag.FlagSet
	flagT := flags.String("t", "", "generator access `token`")
	flagK := flags.String("k", "", "local `pubkey` for signing blocks")
	flagHSMURL := flags.String("hsm-url", "", "hsm `url` for signing blocks (mockhsm if empty)")
	flagHSMToken := flags.String("hsm-token", "", "hsm `acc
ess-token` for connecting to hsm")

	flags.Usage = func() {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()
	if len(args) < 2 {
		fatalln(usage)
	}

	// not a blocksigner
	if *flagK == "" && *flagHSMURL != "" {
		fatalln("error: flag -hsm-url has no effect without -k")
	}

	// TODO(ameets): update when switching to x.509 authorization
	if (*flagHSMURL == "") != (*flagHSMToken == "") {
		fatalln("error: flags -hsm-url and -hsm-token must be given together")
	}

	var blockchainID bc.Hash
	err := blockchainID.UnmarshalText([]byte(args[0]))
	if err != nil {
		fatalln("error: invalid blockchain ID:", err)
	}

	var blockPub []byte
	if *flagK != "" {
		blockPub, err = hex.DecodeString(*flagK)
		if err != nil {
			fatalln("error: unable to decode block pub")
		}
	}

	var conf config.Config
	conf.BlockchainId = &blockchainID
	conf.GeneratorUrl = args[1]
	conf.GeneratorAccessToken = *flagT
	conf.IsSigner = *flagK != ""
	conf.BlockPub = blockPub
	conf.BlockHsmUrl = *flagHSMURL
	conf.BlockHsmAccessToken = *flagHSMToken

	client.BlockchainID = blockchainID.String()
	err = client.Call(context.Background(), "/configure", conf, nil)
	dieOnRPCError(err)
}
*/

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
	io.Copy(os.Stderr, &logbuf)
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(2)
}

func dieOnRPCError(err error, prefixes ...interface{}) {
	if err == nil {
		return
	}

	io.Copy(os.Stderr, &logbuf)

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
	    RootXPubs []chainkd.XPub `json:"root_xpubs"`
		Quorum    int
		Alias     string
		Tags      map[string]interface{}
		ClientToken string `json:"client_token"`
	}
	var ins Ins
	ins.RootXPubs = []chainkd.XPub{xpub}
	ins.Quorum = 1
	ins.Alias = "aa"
	ins.Tags = map[string]interface{}{"test_tag": "v0",}
	ins.ClientToken = args[0]
	responses := make([]interface{}, 50)
	client.Call(context.Background(), "/create-account", &[]Ins{ins,}, &responses)
	//dieOnRPCError(err)
	fmt.Printf("responses:%v\n", responses)
}

func createAsset(client *rpc.Client, args []string) {
	if len(args) != 1 {
		fatalln("error: createAsset takes no args")
	}
	xprv, err := chainkd.NewXPrv(nil)
	if err != nil {
		fatalln("NewXprv error.")
	}
	xpub := xprv.XPub()
	fmt.Printf("xprv:%v\n", xprv)
	fmt.Printf("xpub:%v\n", xpub)
	type Ins struct {
	    RootXPubs []chainkd.XPub `json:"root_xpubs"`
		Quorum    int
		Alias     string
		Tags      map[string]interface{}
		Definition  map[string]interface{}
		ClientToken string `json:"client_token"`
	}
	var ins Ins
	ins.RootXPubs = []chainkd.XPub{xpub}
	ins.Quorum = 1
	ins.Alias = "aa"
	ins.Tags = map[string]interface{}{"test_tag": "v0",}
	ins.Definition = map[string]interface{}{"test_definition": "v0"}
	ins.ClientToken = args[0]
	responses := make([]interface{}, 50)
	client.Call(context.Background(), "/create-asset", &[]Ins{ins,}, &responses)
	//dieOnRPCError(err)
	fmt.Printf("responses:%v\n", responses)
}

func updateAccountTags(client *rpc.Client,args []string){
	if len(args) != 0{
		fatalln("error:updateAccountTags not use args")
	}
	type Ins struct {
	ID    *string
	Alias *string
	Tags  map[string]interface{} `json:"tags"`
}
	var ins Ins
	aa := "1234"
	alias := "asdfg"
	ins.ID = &aa
	ins.Alias = &alias
	ins.Tags = map[string]interface{}{"test_tag": "v0",}
        responses := make([]interface{}, 50)
        client.Call(context.Background(), "/update-account-tags", &[]Ins{ins,}, &responses)
        fmt.Printf("responses:%v\n", responses)
}

func updateAssetTags(client *rpc.Client, args []string){
        if len(args) != 0{
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
        ins.Tags = map[string]interface{}{"test_tag": "v0",}
        responses := make([]interface{}, 50)
        client.Call(context.Background(), "/update-asset-tags", &[]Ins{ins,}, &responses)
        fmt.Printf("responses:%v\n", responses)
}

func createControlProgram(client *rpc.Client, args []string){
        if len(args) != 0{
                fatalln("error:createControlProgram not use args")
        }
	type Ins struct {
	Type   string
	Params stdjson.RawMessage
}
	var ins Ins
	//TODO:undefined arguments to ins
	responses := make([]interface{},50)
        client.Call(context.Background(),"/create-control-program", &[]Ins{ins,}, &responses)
        fmt.Printf("responses:%v\n", responses)
}

func createAccountReceiver(client *rpc.Client, args []string){
        if len(args) != 0{
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
	responses := make([]interface{},50)
        client.Call(context.Background(),"/create-Account-Receiver", &[]Ins{ins,}, &responses)
        fmt.Printf("responses:%v\n", responses)
}

func createTxFeed(client *rpc.Client, args []string){
        if len(args) != 1{
                fatalln("error:createTxFeed take no arguments")
        }
	type In struct {
	Alias  string
	Filter string
	ClientToken string `json:"client_token"`
}
	var in In
	in.Alias = "asdfgh"
	in.Filter = "zxcvbn"
	in.ClientToken = args[0]
	client.Call(context.Background(),"/create-transaction-feed",&[]In{in,},nil)
}

func getTxFeed(client *rpc.Client, args []string){
	if len(args) != 0{
		fatalln("error:getTxFeed not use args")
	}
	type In struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}
	var in In
	in.Alias = "qwerty"
	in.ID = "123456"
	client.Call(context.Background(),"/get-transaction-feed",&[]In{in,},nil)
}

func updateTxFeed(client *rpc.Client, args []string){
        if len(args) != 0{
                fatalln("error:updateTxFeed not use args")
        }
        type In struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}
	var in In
	in.ID = "123456"
	in.Alias = "qwerty"
	client.Call(context.Background(),"/update-transaction-feed",&[]In{in,},nil)
}

func deleteTxFeed(client *rpc.Client, args []string){
	if len(args) != 0{
		fatalln("error:deleteTxFeed not use args")
	}
	type In struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}
	var in In
        in.ID = "123456"
        in.Alias = "qwerty"
        client.Call(context.Background(),"/delete-transaction-feed",&[]In{in,},nil)
}
