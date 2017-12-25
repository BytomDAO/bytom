package commands

import (
	"os"

	"github.com/spf13/cobra"
)

var isMiningCmd = &cobra.Command{
	Use:   "is-mining",
	Short: "If client is actively mining new blocks",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/is-mining")
		if exitCode != Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}
