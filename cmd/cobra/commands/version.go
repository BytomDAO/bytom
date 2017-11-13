package commands

import (
	"runtime"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var version = "0.1.3"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Bytomcli",
	Long:  `All software has versions. This is Bytomcli's.`,
	Run: func(cmd *cobra.Command, args []string) {
		jww.FEEDBACK.Printf("Bytomcli v%s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
	},
}
