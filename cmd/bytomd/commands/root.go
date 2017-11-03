package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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
		config.SetRoot(config.RootDir)
		cfg.EnsureRoot(config.RootDir)
		return nil
	},
}
