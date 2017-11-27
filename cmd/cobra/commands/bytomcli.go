package commands

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/blockchain"
	"github.com/bytom/blockchain/rpc"
)

// commandError is an error used to signal different error situations in command handling.
type commandError struct {
	s         string
	userError bool
}

func (c commandError) Error() string {
	return c.s
}

func (c commandError) isUserError() bool {
	return c.userError
}

func newUserError(a ...interface{}) commandError {
	return commandError{s: fmt.Sprintln(a...), userError: true}
}

func newSystemError(a ...interface{}) commandError {
	return commandError{s: fmt.Sprintln(a...), userError: false}
}

func newSystemErrorF(format string, a ...interface{}) commandError {
	return commandError{s: fmt.Sprintf(format, a...), userError: false}
}

// Catch some of the obvious user errors from Cobra.
// We don't want to show the usage message for every error.
// The below may be to generic. Time will show.
var userErrorRegexp = regexp.MustCompile("argument|flag|shorthand")

func isUserError(err error) bool {
	if cErr, ok := err.(commandError); ok && cErr.isUserError() {
		return true
	}

	return userErrorRegexp.MatchString(err.Error())
}

// BytomcliCmd is Bytomcli's root command.
// Every other command attached to BytomcliCmd is a child command to it.
var BytomcliCmd = &cobra.Command{
	Use:   "bytomcli",
	Short: "Bytomcli is a commond line client for bytom core (a.k.a. bytomd)",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Usage()
		}
	},
}

// Execute adds all child commands to the root command BytomcliCmd and sets flags appropriately.
func Execute() {

	AddCommands()

	if c, err := BytomcliCmd.ExecuteC(); err != nil {
		if isUserError(err) {
			c.Println("")
			c.Println(c.UsageString())
		}

		os.Exit(-1)
	}
}

// AddCommands adds child commands to the root command BytomcliCmd.
func AddCommands() {
	BytomcliCmd.AddCommand(createAccountCmd)
	BytomcliCmd.AddCommand(bindAccountCmd)
	BytomcliCmd.AddCommand(listAccountsCmd)

	BytomcliCmd.AddCommand(createAssetCmd)
	BytomcliCmd.AddCommand(bindAssetCmd)
	BytomcliCmd.AddCommand(listAssetsCmd)

	BytomcliCmd.AddCommand(blockHeightCmd)
	BytomcliCmd.AddCommand(blockHashCmd)
	BytomcliCmd.AddCommand(getBlockByHashCmd)
	BytomcliCmd.AddCommand(getBlockHeaderByHashCmd)
	BytomcliCmd.AddCommand(getBlockTransactionsCountByHashCmd)
	BytomcliCmd.AddCommand(getBlockByHeightCmd)
	BytomcliCmd.AddCommand(getBlockTransactionsCountByHeightCmd)

	BytomcliCmd.AddCommand(createKeyCmd)
	BytomcliCmd.AddCommand(deleteKeyCmd)
	BytomcliCmd.AddCommand(listKeysCmd)

	BytomcliCmd.AddCommand(isMiningCmd)

	BytomcliCmd.AddCommand(netInfoCmd)
	BytomcliCmd.AddCommand(netListeningCmd)
	BytomcliCmd.AddCommand(peerCountCmd)
	BytomcliCmd.AddCommand(netSyncingCmd)

	BytomcliCmd.AddCommand(gasRateCmd)

	BytomcliCmd.AddCommand(createAccessTokenCmd)
	BytomcliCmd.AddCommand(listAccessTokenCmd)
	BytomcliCmd.AddCommand(deleteAccessTokenCmd)
	BytomcliCmd.AddCommand(checkAccessTokenCmd)

	BytomcliCmd.AddCommand(versionCmd)
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
		jww.ERROR.Println("loading TLS cert:", err)
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
