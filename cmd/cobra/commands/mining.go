package commands

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/blockchain"
)

var isMiningCmd = &cobra.Command{
	Use:   "is-mining",
	Short: "If client is actively mining new blocks",
	Run: func(cmd *cobra.Command, args []string) {
		var rawResponse []byte
		var response blockchain.Response
		client := mustRPCClient()
		client.Call(context.Background(), "/is-mining", nil, &rawResponse)

		if err := json.Unmarshal(rawResponse, &response); err != nil {
			jww.ERROR.Println(err)
			return
		}

		if response.Status == blockchain.SUCCESS {
			data := response.Data
			res, err := strconv.ParseBool(data[0])
			if err != nil {
				jww.ERROR.Println("Fail to parse response data")
				return
			}
			jww.FEEDBACK.Printf("is mining: %v\n", res)
			return
		}
		jww.ERROR.Println(response.Msg)
	},
}
