package commands

import (
	"context"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/crypto/ed25519/chainkd"
)

// AssetIns is used for asset related request.
type AssetIns struct {
	RootXPubs   []chainkd.XPub         `json:"root_xpubs"`
	Quorum      int                    `json:"quorum"`
	Alias       string                 `json:"alias"`
	Tags        map[string]interface{} `json:"tags"`
	Definition  map[string]interface{} `json:"definition"`
	ClientToken string                 `json:"client_token"`
}

var createAssetCmd = &cobra.Command{
	Use:   "create-asset",
	Short: "Create an asset",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println("create-asset args invalid\nUsage: create-asset [asset]")
			return
		}

		xprv, err := chainkd.NewXPrv(nil)
		if err != nil {
			jww.ERROR.Println("NewXprv error")
			return
		}

		xPrv, err := xprv.MarshalText()
		if err != nil {
			jww.ERROR.Println("Fail to marshal xPriv")
			return
		}
		jww.FEEDBACK.Printf("xprv: %v\n", string(xPrv))
		xpub := xprv.XPub()
		xPub, _ := xpub.MarshalText()
		jww.FEEDBACK.Printf("xpub: %v\n", xPub)

		var ins AssetIns
		ins.RootXPubs = []chainkd.XPub{xpub}
		ins.Quorum = 1
		ins.Alias = args[0]
		ins.Tags = map[string]interface{}{"test_tag": "v0"}
		ins.Definition = map[string]interface{}{}
		ins.ClientToken = args[0]

		assets := make([]query.AnnotatedAsset, 1)

		client := mustRPCClient()
		client.Call(context.Background(), "/create-asset", &[]AssetIns{ins}, &assets)

		jww.FEEDBACK.Printf("responses: %v\n", assets)
		jww.FEEDBACK.Printf("asset id: %v\n", assets[0].ID.String())
	},
}

var bindAssetCmd = &cobra.Command{
	Use:   "bind-asset",
	Short: "Bind asset",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			jww.ERROR.Println("bind-asset needs 2 args\nUsage: bind-asset [asset name] [asset xpub]")
			return
		}

		var xpub chainkd.XPub
		if err := xpub.UnmarshalText([]byte(args[1])); err != nil {
			jww.ERROR.Printf("xpub unmarshal error: %v\n", xpub)
		}
		jww.FEEDBACK.Printf("xpub: %v\n", xpub)

		var ins AssetIns
		ins.RootXPubs = []chainkd.XPub{xpub}
		ins.Quorum = 1
		ins.Alias = args[0]
		ins.Tags = map[string]interface{}{"test_tag": "v0"}
		ins.Definition = map[string]interface{}{}
		ins.ClientToken = args[0]

		assets := make([]query.AnnotatedAsset, 1)

		client := mustRPCClient()
		client.Call(context.Background(), "/bind-asset", &[]AssetIns{ins}, &assets)

		jww.FEEDBACK.Printf("responses: %v\n", assets)
		jww.FEEDBACK.Printf("asset id: %v\n", assets[0].ID.String())
	},
}

var listAssetsCmd = &cobra.Command{
	Use:   "list-assets",
	Short: "List the existing assets",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			jww.ERROR.Println("list-assets takes no args")
			return
		}

		var in requestQuery

		responses := make([]interface{}, 0)

		client := mustRPCClient()
		client.Call(context.Background(), "/list-assets", in, &responses)

		if len(responses) == 0 {
			jww.FEEDBACK.Println("Empty assets")
		}

		for idx, item := range responses {
			jww.FEEDBACK.Println(idx, ": ", item)
		}
	},
}
