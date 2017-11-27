package commands

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/blockchain"
)

var gasRateCmd = &cobra.Command{
	Use:   "gas-rate",
	Short: "Print the current gas rate",
	Run: func(cmd *cobra.Command, args []string) {
		var rawResponse []byte
		var response blockchain.Response

		client := mustRPCClient()
		client.Call(context.Background(), "/gas-rate", nil, &rawResponse)

		if err := json.Unmarshal(rawResponse, &response); err != nil {
			jww.ERROR.Println(err)
			return
		}

		if response.Status == blockchain.SUCCESS {
			data := response.Data
			i, err := strconv.ParseInt(data[0], 16, 64)
			if err != nil {
				jww.ERROR.Println("Fail to parse response data")
				return
			}
			jww.FEEDBACK.Printf("gas rate: %v\n", i)
			return
		}
		jww.ERROR.Println(response.Msg)
	},
}
