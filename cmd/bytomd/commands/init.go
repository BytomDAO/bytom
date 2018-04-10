package commands

import (
	"github.com/spf13/cobra"

	cfg "github.com/bytom/config"
)

var initFilesCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize blockchain",
	Run:   initFiles,
}

func init() {
	initFilesCmd.Flags().String("chain_id", config.ChainID, "Select [mainnet] or [testnet]")

	RootCmd.AddCommand(initFilesCmd)
}

func initFiles(cmd *cobra.Command, args []string) {
	if config.ChainID == "mainnet" {
		cfg.EnsureRoot(config.RootDir, "mainnet")
	} else {
		cfg.EnsureRoot(config.RootDir, "testnet")
	}
}
