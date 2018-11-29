package commands

import (
	"encoding/hex"
	"os"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/crypto/ed25519/chainkd"
	chainjson "github.com/bytom/encoding/json"
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
		}{Alias: args[0], Password: args[1]}

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
			XPub     chainkd.XPub `json:"xpub"`
		}{XPub: *xpub, Password: args[1]}

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

var updateKeyAliasCmd = &cobra.Command{
	Use:   "update-key-alias <xpub> <new-alias>",
	Short: "Update key alias",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		xpub := new(chainkd.XPub)
		if err := xpub.UnmarshalText([]byte(args[0])); err != nil {
			jww.ERROR.Println("update-key-alias xpub not valid:", err)
			os.Exit(util.ErrLocalExe)
		}

		ins := struct {
			XPub     chainkd.XPub `json:"xpub"`
			NewAlias string       `json:"new_alias"`
		}{XPub: *xpub, NewAlias: args[1]}

		if _, exitCode := util.ClientCall("/update-key-alias", &ins); exitCode != util.Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Println("Successfully update key alias")
	},
}

var resetKeyPwdCmd = &cobra.Command{
	Use:   "reset-key-password <xpub> <old-password> <new-password>",
	Short: "Reset key password",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		xpub := new(chainkd.XPub)
		if err := xpub.UnmarshalText([]byte(args[0])); err != nil {
			jww.ERROR.Println("reset-key-password args not valid:", err)
			os.Exit(util.ErrLocalExe)
		}

		ins := struct {
			XPub        chainkd.XPub `json:"xpub"`
			OldPassword string       `json:"old_password"`
			NewPassword string       `json:"new_password"`
		}{XPub: *xpub, OldPassword: args[1], NewPassword: args[2]}

		data, exitCode := util.ClientCall("/reset-key-password", &ins)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var checkKeyPwdCmd = &cobra.Command{
	Use:   "check-key-password <xpub> <password>",
	Short: "check key password",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		xpub := new(chainkd.XPub)
		if err := xpub.UnmarshalText([]byte(args[0])); err != nil {
			jww.ERROR.Println("check-key-password args not valid:", err)
			os.Exit(util.ErrLocalExe)
		}

		ins := struct {
			XPub     chainkd.XPub `json:"xpub"`
			Password string       `json:"password"`
		}{XPub: *xpub, Password: args[1]}

		data, exitCode := util.ClientCall("/check-key-password", &ins)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var signMsgCmd = &cobra.Command{
	Use:   "sign-message <address> <message> <password>",
	Short: "sign message to generate signature",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		message, err := hex.DecodeString(args[1])
		if err != nil {
			jww.ERROR.Println("sign-message args not valid:", err)
			os.Exit(util.ErrLocalExe)
		}

		var req = struct {
			Address  string             `json:"address"`
			Message  chainjson.HexBytes `json:"message"`
			Password string             `json:"password"`
		}{Address: args[0], Message: message, Password: args[2]}

		data, exitCode := util.ClientCall("/sign-message", &req)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var verifyMsgCmd = &cobra.Command{
	Use:   "verify-message <address> <xpub> <message> <signature>",
	Short: "verify signature for specified message",
	Args:  cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		xpub := chainkd.XPub{}
		if err := xpub.UnmarshalText([]byte(args[1])); err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalExe)
		}

		message, err := hex.DecodeString(args[2])
		if err != nil {
			jww.ERROR.Println("sign-message args not valid:", err)
			os.Exit(util.ErrLocalExe)
		}

		var req = struct {
			Address     string             `json:"address"`
			DerivedXPub chainkd.XPub       `json:"derived_xpub"`
			Message     chainjson.HexBytes `json:"message"`
			Signature   string             `json:"signature"`
		}{Address: args[0], DerivedXPub: xpub, Message: message, Signature: args[3]}

		data, exitCode := util.ClientCall("/verify-message", &req)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}
