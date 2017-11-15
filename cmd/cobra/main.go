package main

import (
	"runtime"

	cmd "github.com/bytom/cmd/cobra/commands"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	cmd.Execute()
}
