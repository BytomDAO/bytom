package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/bytom/types"
	cmn "github.com/tendermint/tmlibs/common"
)

var initFilesCmd = &cobra.Command {
	Use:   "init",
	Short: "Initialize blockchain",
	Run:   initFiles,
}

func init() {
	RootCmd.AddCommand(initFilesCmd)
}

func initFiles(cmd *cobra.Command, args []string) {
	genFile := config.GenesisFile()

	if _, err := os.Stat(genFile); os.IsNotExist(err) {
		genDoc := types.GenesisDoc{
			ChainID: cmn.Fmt("bytom"),
		}

		genDoc.SaveAs(genFile)
	}

	logger.Info("Initialized bytom", "genesis", config.GenesisFile())
}
