package commands

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/bytom/util"
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
