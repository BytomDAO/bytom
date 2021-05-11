package commands

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	cfg "github.com/bytom/bytom/config"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
)

var initFilesCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize blockchain",
	Run:   initFiles,
}

func init() {
	initFilesCmd.Flags().String("chain_id", config.ChainID, "Select [mainnet] or [testnet] or [solonet]")

	RootCmd.AddCommand(initFilesCmd)
}

func initFiles(cmd *cobra.Command, args []string) {
	configFilePath := path.Join(config.RootDir, "config.toml")
	if _, err := os.Stat(configFilePath); !os.IsNotExist(err) {
		log.WithFields(log.Fields{"module": logModule, "config": configFilePath}).Info("Already exists config file.")
		return
	}

	switch config.ChainID {
	case "mainnet", "testnet":
		cfg.EnsureRoot(config.RootDir, config.ChainID)
	default:
		cfg.EnsureRoot(config.RootDir, "solonet")
	}

	//generate the node private key
	keyFilePath := path.Join(config.RootDir, config.PrivateKeyFile)
	if _, err := os.Stat(keyFilePath); os.IsNotExist(err) {
		xprv, err := chainkd.NewXPrv(nil)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Fatal("fail on generate private key")
		}

		if err := ioutil.WriteFile(keyFilePath, []byte(hex.EncodeToString(xprv[:])), 0600); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Fatal("fail on save private key")
		}

		log.WithFields(log.Fields{"pubkey": xprv.XPub()}).Info("success generate private")
	}

	log.WithFields(log.Fields{"module": logModule, "config": configFilePath}).Info("Initialized bytom")
}
