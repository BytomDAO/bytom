package commands

import (
	"os/user"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cmn "github.com/tendermint/tmlibs/common"

	cfg "github.com/bytom/config"
)

var (
	config = cfg.DefaultConfig()
)

var RootCmd = &cobra.Command{
	Use:   "bytomd",
	Short: "Multiple asset management.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := viper.Unmarshal(config)
		if err != nil {
			return err
		}
		paths := strings.SplitN(config.RootDir, "/", 2)
		if len(paths) == 2 && (paths[0] == "~" || paths[0] == "$HOME") {
			usr, err := user.Current()
			if err != nil {
				cmn.Exit("Error: " + err.Error())
			}
			paths[0] = usr.HomeDir
			config.RootDir = paths[0] + "/" + paths[1]
		}
		config.SetRoot(config.RootDir)
		return nil
	},
}
