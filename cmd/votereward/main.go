package main

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tendermint/tmlibs/cli"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/toolbar/common"
	cfg "github.com/bytom/bytom/toolbar/vote_reward/config"
	"github.com/bytom/bytom/toolbar/vote_reward/settlementvotereward"
	"github.com/bytom/bytom/toolbar/vote_reward/synchron"
)

const logModule = "reward"

var (
	rewardStartHeight uint64
	rewardEndHeight   uint64
	configFile        string
)

var RootCmd = &cobra.Command{
	Use:   "reward",
	Short: "distribution of reward.",
	RunE:  runReward,
}

func init() {
	RootCmd.Flags().Uint64Var(&rewardStartHeight, "reward_start_height", 0, "The starting height of the distributive income reward interval, It is a multiple of the pos consensus cycle(100). example: 600")
	RootCmd.Flags().Uint64Var(&rewardEndHeight, "reward_end_height", 0, "The end height of the distributive income reward interval, It is a multiple of the pos consensus cycle(100). example: 1200")
	RootCmd.Flags().StringVar(&configFile, "config_file", "reward.json", "config file. default: reward.json")
}

func runReward(cmd *cobra.Command, args []string) error {
	log.Info("This tool belongs to an open-source project, we can not guarantee this tool is bug-free. Please check the code before using, developers will not be responsible for any asset loss due to bug!")
	startTime := time.Now()
	config := &cfg.Config{}
	if err := cfg.LoadConfigFile(configFile, config); err != nil {
		log.WithFields(log.Fields{"module": logModule, "config": configFile, "error": err}).Fatal("Failded to load config file.")
	}

	if err := consensus.InitActiveNetParams(config.ChainID); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Fatal("Init ActiveNetParams.")
	}
	if rewardStartHeight >= rewardEndHeight || rewardStartHeight%consensus.ActiveNetParams.BlocksOfEpoch != 0 || rewardEndHeight%consensus.ActiveNetParams.BlocksOfEpoch != 0 {
		log.Fatal("Please check the height range, which must be multiple of the number of block rounds.")
	}

	db, err := common.NewMySQLDB(config.MySQLConfig)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Fatal("Failded to initialize mysql db.")
	}

	db.LogMode(true)

	keeper, err := synchron.NewChainKeeper(db, config, rewardEndHeight)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Fatal("Failded to initialize NewChainKeeper.")
	}

	if err := keeper.SyncBlock(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Fatal("Failded to sync block.")
	}

	s := settlementvotereward.NewSettlementReward(db, config, rewardStartHeight, rewardEndHeight)

	if err := s.Settlement(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Fatal("Settlement vote rewards failure.")
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"duration": time.Since(startTime),
	}).Info("Settlement vote reward complete")

	return nil
}

func main() {
	cmd := cli.PrepareBaseCmd(RootCmd, "REWARD", "./")
	cmd.Execute()
}
