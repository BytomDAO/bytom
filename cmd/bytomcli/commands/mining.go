package commands

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/bytom/util"
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

var setMiningCmd = &cobra.Command{
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

		if _, exitCode := util.ClientCall("/set-mining", miningInfo); exitCode != util.Success {
			os.Exit(exitCode)
		}

		if isMining {
			jww.FEEDBACK.Println("start mining success")
		} else {
			jww.FEEDBACK.Println("stop mining success")
		}
	},
}
