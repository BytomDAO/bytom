package commands

import (
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var gasRateCmd = &cobra.Command{
	Use:   "gas-rate",
	Short: "Print the current gas rate",
	Run: func(cmd *cobra.Command, args []string) {
		if data := clientCall("/gas-rate", nil); data != nil {
			i, err := strconv.ParseInt(data[0], 16, 64)
			if err != nil {
				jww.ERROR.Println("Fail to parse response data")
				return
			}
			jww.FEEDBACK.Printf("gas rate: %v\n", i)
		}
	},
}
