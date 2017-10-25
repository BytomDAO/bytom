package main

import (
	"github.com/bytom/cmd/bytomd/commands"
	"github.com/tendermint/tmlibs/cli"
	"os"
)

func main() {
	cmd := cli.PrepareBaseCmd(commands.RootCmd, "TM", os.ExpandEnv("./.bytomd"))
	cmd.Execute()
}
