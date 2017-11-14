package commands

import (
	"context"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/encoding/json"
)

var createAssetCmd = &cobra.Command{
	Use:   "create-asset",
	Short: "Create an asset",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println(`
create-asset args invalid

Usage: create-asset [asset]`)
			return
		}

		xprv, err := chainkd.NewXPrv(nil)
		if err != nil {
			jww.ERROR.Println("NewXprv error")
		}

		xprv_, _ := xprv.MarshalText()
		xpub := xprv.XPub()
		jww.FEEDBACK.Printf("xprv: %v\n", string(xprv_))
		xpub_, _ := xpub.MarshalText()
		jww.FEEDBACK.Printf("xpub: %v\n", xpub_)

		type Ins struct {
			RootXPubs   []chainkd.XPub `json:"root_xpubs"`
			Quorum      int
			Alias       string
			Tags        map[string]interface{}
			Definition  map[string]interface{}
			ClientToken string `json:"client_token"`
		}

		var ins Ins
		ins.RootXPubs = []chainkd.XPub{xpub}
		ins.Quorum = 1
		ins.Alias = args[0]
		ins.Tags = map[string]interface{}{"test_tag": "v0"}
		ins.Definition = map[string]interface{}{}
		ins.ClientToken = args[0]

		assets := make([]query.AnnotatedAsset, 1)

		client := mustRPCClient()
		client.Call(context.Background(), "/create-asset", &[]Ins{ins}, &assets)

		jww.FEEDBACK.Printf("responses: %v\n", assets)
		jww.FEEDBACK.Printf("asset id: %v\n", assets[0].ID.String())
	},
}

var listAssetCmd = &cobra.Command{
	Use:   "list-asset",
	Short: "List asset",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			jww.ERROR.Println("list-asset takes no args")
			return
		}

		type requestQuery struct {
			Filter       string        `json:"filter,omitempty"`
			FilterParams []interface{} `json:"filter_params,omitempty"`
			SumBy        []string      `json:"sum_by,omitempty"`
			PageSize     int           `json:"page_size"`
			AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
			Timeout      json.Duration `json:"timeout"`
			After        string        `json:"after"`
			StartTimeMS  uint64        `json:"start_time,omitempty"`
			EndTimeMS    uint64        `json:"end_time,omitempty"`
			TimestampMS  uint64        `json:"timestamp,omitempty"`
			Type         string        `json:"type"`
			Aliases      []string      `json:"aliases,omitempty"`
		}
		var in requestQuery

		responses := make([]interface{}, 0)

		client := mustRPCClient()
		client.Call(context.Background(), "/list-assets", in, &responses)

		if len(responses) > 0 {
			for idx, item := range responses {
				jww.FEEDBACK.Println(idx, ": ", item)
			}
		} else {
			jww.FEEDBACK.Println("Empty assets")
		}
	},
}

var bindAssetCmd = &cobra.Command{
	Use:   "bind-asset",
	Short: "Bind asset",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			jww.ERROR.Println(`
bind-asset needs 2 args

Usage: bind-asset [asset name] [asset xpub]`)
			return
		}

		var xpub chainkd.XPub
		err := xpub.UnmarshalText([]byte(args[1]))
		if err != nil {
			jww.ERROR.Printf("xpub unmarshal error: %v\n", xpub)
		}
		jww.FEEDBACK.Printf("xpub: %v\n", xpub)

		type Ins struct {
			RootXPubs   []chainkd.XPub `json:"root_xpubs"`
			Quorum      int
			Alias       string
			Tags        map[string]interface{}
			Definition  map[string]interface{}
			ClientToken string `json:"client_token"`
		}

		var ins Ins
		ins.RootXPubs = []chainkd.XPub{xpub}
		ins.Quorum = 1
		ins.Alias = args[0]
		ins.Tags = map[string]interface{}{"test_tag": "v0"}
		ins.Definition = map[string]interface{}{}
		ins.ClientToken = args[0]

		assets := make([]query.AnnotatedAsset, 1)

		client := mustRPCClient()
		client.Call(context.Background(), "/create-asset", &[]Ins{ins}, &assets)

		jww.FEEDBACK.Printf("responses: %v\n", assets)
		jww.FEEDBACK.Printf("asset id: %v\n", assets[0].ID.String())
	},
}
