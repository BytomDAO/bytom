package commands

import (
	"os"
	"strings"

	"github.com/bytom/util"
	"github.com/spf13/cobra"
)

var isMiningCmd = &cobra.Command{
	Use:   "is-mining",
	Short: "If client is actively mining new blocks",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/is-mining")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var miningCmd = &cobra.Command{
	Use:   "set-mining <true or false>",
	Short: "start or stop mining",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		param := strings.ToLower(args[0])
		isMining := false
		switch param {
		case "true":
			isMining = true
		default:
			isMining = false
		}
		miningInfo := &struct {
			IsMining bool `json:"is_mining"`
		}{IsMining: isMining}
		data, exitCode := util.ClientCall("/set-mining", miningInfo)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}
