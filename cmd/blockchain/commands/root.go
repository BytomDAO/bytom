package commands

import (
	"os"

	"github.com/spf13/cobra"

//	tmflags "github.com/blockchain/cmd/blockchain/commands/flags"
	"github.com/tendermint/tmlibs/log"
)

var (
	logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "main")
)

func init() {
	RootCmd.PersistentFlags().String("log_level", "*:info", "Log level")
}

var RootCmd = &cobra.Command{
	Use:   "blockchain",
	Short: "blockchain in Go",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		/*logger, err := tmflags.ParseLogLevel("*:info", logger)
		if err != nil {
			return err
		}*/
		return nil
	},
}
