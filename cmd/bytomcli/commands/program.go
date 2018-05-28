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
