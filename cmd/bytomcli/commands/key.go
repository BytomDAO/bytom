package commands

import (
	"bytes"
	"encoding/hex"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/tendermint/go-wire/data/base58"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/util"
)

var createKeyCmd = &cobra.Command{
	Use:   "create-key <alias> <password>",
	Short: "Create a key",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var key = struct {
			Alias    string `json:"alias"`
			Password string `json:"password"`
		}{Alias: args[0], Password: "123456"}

		data, exitCode := util.ClientCall("/create-key", &key)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var deleteKeyCmd = &cobra.Command{
	Use:   "delete-key <xpub> <password>",
	Short: "Delete a key",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		xpub := new(chainkd.XPub)
		if err := xpub.UnmarshalText([]byte(args[0])); err != nil {
			jww.ERROR.Println("delete-key:", err)
			os.Exit(util.ErrLocalExe)
		}

		var key = struct {
			Password string
			XPub     chainkd.XPub `json:"xpubs"`
		}{XPub: *xpub, Password: "123456"}

		if _, exitCode := util.ClientCall("/delete-key", &key); exitCode != util.Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Println("Successfully delete key")
	},
}

var listKeysCmd = &cobra.Command{
	Use:   "list-keys",
	Short: "List the existing keys",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/list-keys")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSONList(data)
	},
}

var exportPrivateCmd = &cobra.Command{
	Use:   "export-private-key <xpub> <password>",
	Short: "Export the private key",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		type Key struct {
			Password string
			XPub     chainkd.XPub
		}
		var key Key
		xpub := new(chainkd.XPub)
		rawPub, err := hex.DecodeString(args[0])
		if err != nil {
			jww.ERROR.Println("error: export-private-key args not vaild", err)
		}
		copy(xpub[:], rawPub)

		key.XPub = *xpub
		key.Password = args[1]

		data, exitCode := util.ClientCall("/export-private-key", &key)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var importPrivateCmd = &cobra.Command{
	Use:   "import-private-key <alias> <private key> <index> <password>",
	Short: "Import the private key",
	Args:  cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		type Key struct {
			Alias    string
			Password string
			XPrv     chainkd.XPrv
			Index    uint64
		}

		privHash, err := base58.Decode(args[1])
		if err != nil {
			jww.ERROR.Println("wif priv decode error")
			os.Exit(util.ErrLocalExe)
		}
		if len(privHash) != 68 {
			jww.ERROR.Println("wif priv length error")
			os.Exit(util.ErrLocalExe)
		}
		var hashed [32]byte

		sha3pool.Sum256(hashed[:], privHash[:64])

		if res := bytes.Compare(hashed[:4], privHash[64:]); res != 0 {
			jww.ERROR.Println("wif priv hash error")
			os.Exit(util.ErrLocalExe)
		}

		var key Key
		key.Alias = args[0]
		key.Password = args[3]
		key.Index, _ = strconv.ParseUint(args[2], 10, 64)
		copy(key.XPrv[:], privHash[:64])

		data, exitCode := util.ClientCall("/import-private-key", &key)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}
