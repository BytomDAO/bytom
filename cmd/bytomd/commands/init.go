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

	if _, err := os.Stat(genFile); os.IsNotExist(err) {
		xprv, errKey := chainkd.NewXPrv(nil)
		if errKey != nil {
			log.WithField("error", errKey).Error("Spawn node's key failed.")
		} else {
			genDoc := types.GenesisDoc{
				ChainID:    cmn.Fmt("bytom"),
				PrivateKey: hex.EncodeToString(xprv.Bytes()),
			}
			genDoc.SaveAs(genFile)
			log.WithField("genesis", config.GenesisFile()).Info("Initialized bytom")
		}
	} else {
			log.WithField("genesis", config.GenesisFile()).Info("Already exits config file.")
	}

}
