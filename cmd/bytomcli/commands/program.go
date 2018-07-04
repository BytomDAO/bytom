package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/bytom/util"
)

var decodeProgCmd = &cobra.Command{
	Use:   "decode-program <program>",
	Short: "decode program to instruction and data",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var req = struct {
			Program string `json:"program"`
		}{Program: args[0]}

		data, exitCode := util.ClientCall("/decode-program", &req)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var sha3Cmd = &cobra.Command{
	Use:   "sha3 <data>",
	Short: "calculate hash by sha3",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var req = struct {
			Data string `json:"data"`
		}{Data: args[0]}

		data, exitCode := util.ClientCall("/sha3", &req)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var sha256Cmd = &cobra.Command{
	Use:   "sha256 <data>",
	Short: "calculate hash by sha256",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var req = struct {
			Data string `json:"data"`
		}{Data: args[0]}

		data, exitCode := util.ClientCall("/sha256", &req)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}
