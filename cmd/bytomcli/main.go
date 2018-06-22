package main

import (
	"runtime"

	cmd "github.com/bytom/bytom/cmd/bytomcli/commands"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	cmd.Execute()
}
