package commands

import (
	"encoding/hex"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	cmn "github.com/tendermint/tmlibs/common"

	cfg "github.com/bytom/config"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/types"
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

	genFile := config.GenesisFile()
	if _, err := os.Stat(genFile); !os.IsNotExist(err) {
		log.WithField("genesis", config.GenesisFile()).Info("Already exists config file.")
		return
	}
	xprv, err := chainkd.NewXPrv(nil)
	if err != nil {
		log.WithField("error", err).Error("Spawn node's key failed.")
		return
	}
	genDoc := types.GenesisDoc{
		ChainID:    cmn.Fmt(config.ChainID),
		PrivateKey: hex.EncodeToString(xprv.Bytes()),
	}
	genDoc.SaveAs(genFile)
	log.WithField("genesis", config.GenesisFile()).Info("Initialized bytom")
}
