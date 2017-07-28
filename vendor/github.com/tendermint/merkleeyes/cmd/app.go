package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tendermint/abci/server"

	cmn "github.com/tendermint/tmlibs/common"

	application "github.com/tendermint/merkleeyes/app"
)

var (
	address string
	abci    string
	cache   int
)

var startCmd = &cobra.Command{
	Run:   StartServer,
	Use:   "start",
	Short: "Start the MerkleEyes server",
	Long:  `Startup the MerkleEyes ABCi app`,
}

func init() {
	RootCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&address, "address", "l", "unix://data.sock", "MerkleEyes server listen address")
	startCmd.Flags().StringVarP(&abci, "abci", "a", "socket", "socket | grpc")
	startCmd.Flags().IntVarP(&cache, "cache", "c", 0, "database cache size")
}

func StartServer(cmd *cobra.Command, args []string) {
	app := application.NewMerkleEyesApp(dbName, cache)
	server, err := server.NewServer(address, abci, app)

	if err != nil {
		cmn.Exit(err.Error())
	}

	cmn.TrapSignal(func() {
		app.CloseDB()
		server.Stop()
	})
}
