package commands

import (
	"encoding/hex"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	cmn "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"
	"github.com/bytom/blockchain/account"

	cfg "github.com/bytom/config"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/types"
	"github.com/bytom/blockchain/pseudohsm"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize blockchain",
	Run:   doInit,
}

func init() {
	initCmd.Flags().String("chain_id", config.ChainID, "Select [mainnet] or [testnet]")

	RootCmd.AddCommand(initCmd)
}

func doInit(cmd *cobra.Command, args []string) {
	initAccount()
	initFiles()
}

func initAccount() {
	hsm, err := pseudohsm.New(config.KeysDir())
	if err != nil {
		cmn.Exit(cmn.Fmt("initialize HSM failed: %v", err))
	}

	var xpub *pseudohsm.XPub = nil
	if len(hsm.ListKeys()) == 0 {
		if xpub, err = hsm.XCreate("default", "123456"); err != nil {
			cmn.Exit(cmn.Fmt("create default key failed: %v", err))
		}
	}

	walletDB := dbm.NewDB("wallet", config.DBBackend, config.DBDir())
	accountManager := account.NewManager(walletDB, nil)
	if accounts, err := accountManager.ListAccounts(""); err != nil {
		cmn.Exit(cmn.Fmt("check account failed: %v", err))
	} else if len(accounts) == 0 {
		accountManager.Create(nil, []chainkd.XPub{xpub.XPub}, 1, "default", nil)
	}
}

func initFiles() {
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
