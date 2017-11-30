package main

import (
	"os"

	"github.com/tendermint/tmlibs/cli"

	"github.com/bytom/cmd/bytomd/commands"
)

func main() {
	cmd := cli.PrepareBaseCmd(commands.RootCmd, "TM", os.ExpandEnv("./.bytomd"))
	cmd.Execute()
}
