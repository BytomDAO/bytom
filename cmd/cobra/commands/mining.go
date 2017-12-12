package commands

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var isMiningCmd = &cobra.Command{
	Use:   "is-mining",
	Short: "If client is actively mining new blocks",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/is-mining")
		if exitCode != Success {
			os.Exit(exitCode)
		}
		res, err := strconv.ParseBool(data[0])
		if err != nil {
			jww.ERROR.Println("Fail to parse response data")
			os.Exit(ErrLocalUnwrap)
		}
		jww.FEEDBACK.Printf("is mining: %v\n", res)
	},
}
