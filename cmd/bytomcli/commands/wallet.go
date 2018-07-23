package commands

import (
	"os"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/util"
)

var walletInfoCmd = &cobra.Command{
	Use:   "wallet-info",
	Short: "Print the information of wallet",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/wallet-info")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var rescanWalletCmd = &cobra.Command{
	Use:   "rescan-wallet",
	Short: "Trigger to rescan block information into related wallet",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if _, exitCode := util.ClientCall("/rescan-wallet"); exitCode != util.Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Println("Successfully trigger rescanning wallet")
	},
}
