package commands

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var gasRateCmd = &cobra.Command{
	Use:   "gas-rate",
	Short: "Print the current gas rate",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/gas-rate", nil)
		if exitCode != Success {
			os.Exit(exitCode)
		}
		i, err := strconv.ParseInt(data[0], 16, 64)
		if err != nil {
			jww.ERROR.Println("Fail to parse response data")
			os.Exit(ErrLocalUnwrap)
		}
		jww.FEEDBACK.Printf("gas rate: %v\n", i)
	},
}
