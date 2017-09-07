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
			PrivateKey: "27F82582AEFAE7AB151CFB01C48BB6C1A0DA78F9BDDA979A9F70A84D074EB07D3B3069C422E19688B45CBFAE7BB009FC0FA1B1EA86593519318B7214853803C8",
		}
		genDoc.SaveAs(genFile)
	}

	logger.Info("Initialized bytom", "genesis", config.GenesisFile())
}
