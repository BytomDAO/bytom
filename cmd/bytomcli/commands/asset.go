package commands

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/util"
)

func init() {
	createAssetCmd.PersistentFlags().IntVarP(&assetQuorum, "quorom", "q", 1, "quorum must be greater than 0 and less than or equal to the number of signers")
	createAssetCmd.PersistentFlags().StringVarP(&assetToken, "access", "a", "", "access token")
	createAssetCmd.PersistentFlags().StringVarP(&assetTags, "tags", "t", "", "tags")
	createAssetCmd.PersistentFlags().StringVarP(&assetDefiniton, "definition", "d", "", "definition for the asset")

	updateAssetTagsCmd.PersistentFlags().StringVarP(&assetUpdateTags, "tags", "t", "", "tags to add, delete or update")

	listAssetsCmd.PersistentFlags().StringVar(&assetID, "id", "", "ID of asset")
}

var (
	assetID         = ""
	assetQuorum     = 1
	assetToken      = ""
	assetTags       = ""
	assetDefiniton  = ""
	assetUpdateTags = ""
)

var createAssetCmd = &cobra.Command{
	Use:   "create-asset <alias> <xpub(s)>",
	Short: "Create an asset",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		var ins assetIns

		for _, x := range args[1:] {
			xpub := chainkd.XPub{}
			if err := xpub.UnmarshalText([]byte(x)); err != nil {
				jww.ERROR.Println(err)
				os.Exit(util.ErrLocalExe)
			}
			ins.RootXPubs = append(ins.RootXPubs, xpub)
		}

		ins.Quorum = assetQuorum
		ins.Alias = args[0]
		ins.AccessToken = assetToken

		if len(assetTags) != 0 {
			tags := strings.Split(assetTags, ":")
			if len(tags) != 2 {
				jww.ERROR.Println("Invalid tags")
				os.Exit(util.ErrLocalExe)
			}
			ins.Tags = map[string]interface{}{tags[0]: tags[1]}
		}

		if len(assetDefiniton) != 0 {
			definition := strings.Split(assetDefiniton, ":")
			if len(definition) != 2 {
				jww.ERROR.Println("Invalid definition")
				os.Exit(util.ErrLocalExe)
			}
			ins.Definition = map[string]interface{}{definition[0]: definition[1]}
		}

		data, exitCode := util.ClientCall("/create-asset", &ins)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var listAssetsCmd = &cobra.Command{
	Use:   "list-assets",
	Short: "List the existing assets",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		filter := struct {
			ID string `json:"id"`
		}{ID: assetID}

		data, exitCode := util.ClientCall("/list-assets", &filter)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSONList(data)
	},
}

var updateAssetTagsCmd = &cobra.Command{
	Use:   "update-asset-tags <assetID|alias>",
	Short: "Update the asset tags",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		cmd.MarkFlagRequired("tags")
	},
	Run: func(cmd *cobra.Command, args []string) {
		var updateTag = struct {
			AssetInfo string                 `json:"asset_info"`
			Tags      map[string]interface{} `json:"tags"`
		}{}

		if len(assetUpdateTags) != 0 {
			tags := strings.Split(assetUpdateTags, ":")
			if len(tags) != 2 {
				jww.ERROR.Println("Invalid tags")
				os.Exit(util.ErrLocalExe)
			}
			updateTag.Tags = map[string]interface{}{tags[0]: tags[1]}
		}

		updateTag.AssetInfo = args[0]
		if _, exitCode := util.ClientCall("/update-asset-tags", &updateTag); exitCode != util.Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Println("Successfully update asset tags")
	},
}

var updateAssetAliasCmd = &cobra.Command{
	Use:   "update-asset-alias <oldAlias> <newAlias>",
	Short: "Update the asset alias",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var updateAlias = struct {
			OldAlias string `json:"old_alias"`
			NewAlias string `json:"new_alias"`
		}{OldAlias: args[0], NewAlias: args[1]}

		if _, exitCode := util.ClientCall("/update-asset-alias", &updateAlias); exitCode != util.Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Println("Successfully update asset alias")
	},
}
