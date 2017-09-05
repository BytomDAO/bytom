package main

import (
	"os"

	"github.com/bytom/cmd/bytom/commands"
	"github.com/tendermint/tmlibs/cli"
)

func main() {
	cmd := cli.PrepareBaseCmd(commands.RootCmd, "TM", os.ExpandEnv("./.bytom"))
	cmd.Execute()
}
