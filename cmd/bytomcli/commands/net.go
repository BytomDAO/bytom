package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/bytom/bytom/util"
)

var netInfoCmd = &cobra.Command{
	Use:   "net-info",
	Short: "Print the summary of network",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/net-info")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}
