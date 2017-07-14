package main

import (
	"os"

	"github.com/node_p2p/node_p2p/commands"
	"github.com/tendermint/tmlibs/cli"
)

func main() {
	cmd := cli.PrepareBaseCmd(commands.RootCmd, "TM", os.ExpandEnv("./.node"))
	cmd.Execute()
}
