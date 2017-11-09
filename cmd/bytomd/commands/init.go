package commands

import (
	"os"
	"encoding/hex"

	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/bytom/types"
	"github.com/bytom/crypto/ed25519/chainkd"
)

var initFilesCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize blockchain",
	Run:   initFiles,
}

func init() {
	RootCmd.AddCommand(initFilesCmd)
}

func initFiles(cmd *cobra.Command, args []string) {
	genFile := config.GenesisFile()
	if _, err := os.Stat(genFile); !os.IsNotExist(err) {
		log.WithField("genesis", config.GenesisFile()).Info("Already exits config file.")
		return
	}
	xprv, err := chainkd.NewXPrv(nil)
	if err != nil {
		log.WithField("error", err).Error("Spawn node's key failed.")
		return
	}
	genDoc := types.GenesisDoc{
		ChainID:    cmn.Fmt("bytom"),
		PrivateKey: hex.EncodeToString(xprv.Bytes()),
	}
	genDoc.SaveAs(genFile)
	log.WithField("genesis", config.GenesisFile()).Info("Initialized bytom")
}
