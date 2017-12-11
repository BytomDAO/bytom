package commands

import (
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var isMiningCmd = &cobra.Command{
	Use:   "is-mining",
	Short: "If client is actively mining new blocks",
	Run: func(cmd *cobra.Command, args []string) {
		if data := clientCall("/is-mining"); data != nil {
			res, err := strconv.ParseBool(data[0])
			if err != nil {
				jww.ERROR.Println("Fail to parse response data")
				return
			}
			jww.FEEDBACK.Printf("is mining: %v\n", res)
		}
	},
}
