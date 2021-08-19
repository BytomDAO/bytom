package commands

import (
	"encoding/hex"
	"os"
	"strconv"
	"unicode"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	chainjson "github.com/bytom/bytom/encoding/json"
	"github.com/bytom/bytom/util"
)

var getBlockHashCmd = &cobra.Command{
	Use:   "get-block-hash",
	Short: "Get the hash of most recent block",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("get-block-hash")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var getBlockCountCmd = &cobra.Command{
	Use:   "get-block-count",
	Short: "Get the number of most recent block",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/get-block-count")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var getBlockCmd = &cobra.Command{
	Use:   "get-block <hash> | <height>",
	Short: "Get a whole block matching the given hash or height",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var hash chainjson.HexBytes
		var height uint64
		var err error
		isNumber := false

		for _, ch := range args[0] {
			// check whether the char is hex digit
			if !(unicode.IsNumber(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				jww.ERROR.Printf("Invalid value for hash or height")
				os.Exit(util.ErrLocalExe)
			}

			if !unicode.IsNumber(ch) {
				isNumber = true
			}
		}

		if isNumber {
			if len(args[0]) != 64 {
				jww.ERROR.Printf("Invalid hash length")
				os.Exit(util.ErrLocalExe)
			}

			hash, err = hex.DecodeString(args[0])
			if err != nil {
				jww.ERROR.Println(err)
				os.Exit(util.ErrLocalExe)
			}
		} else {
			height, err = strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				jww.ERROR.Printf("Invalid height value")
				os.Exit(util.ErrLocalExe)
			}
		}

		blockReq := &struct {
			BlockHeight uint64             `json:"block_height"`
			BlockHash   chainjson.HexBytes `json:"block_hash"`
		}{BlockHeight: height, BlockHash: hash}

		data, exitCode := util.ClientCall("/get-block", blockReq)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var getBlockHeaderCmd = &cobra.Command{
	Use:   "get-block-header <hash> | <height>",
	Short: "Get the header of a block matching the given hash or height",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var hash chainjson.HexBytes
		var height uint64
		var err error
		isNumber := false

		for _, ch := range args[0] {
			// check whether the char is hex digit
			if !(unicode.IsNumber(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				jww.ERROR.Printf("Invalid value for hash or height")
				os.Exit(util.ErrLocalExe)
			}

			if !unicode.IsNumber(ch) {
				isNumber = true
			}
		}

		if isNumber {
			if len(args[0]) != 64 {
				jww.ERROR.Printf("Invalid hash length")
				os.Exit(util.ErrLocalExe)
			}

			hash, err = hex.DecodeString(args[0])
			if err != nil {
				jww.ERROR.Println(err)
				os.Exit(util.ErrLocalExe)
			}
		} else {
			height, err = strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				jww.ERROR.Printf("Invalid height value")
				os.Exit(util.ErrLocalExe)
			}
		}

		req := &struct {
			BlockHeight uint64             `json:"block_height"`
			BlockHash   chainjson.HexBytes `json:"block_hash"`
		}{BlockHeight: height, BlockHash: hash}

		data, exitCode := util.ClientCall("/get-block-header", req)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}
