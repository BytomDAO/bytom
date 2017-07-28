package cmd

import (
	"fmt"
	"path"

	"github.com/spf13/cobra"

	cmn "github.com/tendermint/tmlibs/common"
	db "github.com/tendermint/tmlibs/db"

	"github.com/tendermint/merkleeyes/iavl"
)

var (
	dbDir     string
	verbose   bool
	cacheSize int
)

var dumpCmd = &cobra.Command{
	Run:   DumpDatabase,
	Use:   "dump",
	Short: "Dump a database",
	Long:  `Dump all of the data for an underlying persistent database`,
}

func init() {
	RootCmd.AddCommand(dumpCmd)
	dumpCmd.Flags().StringVarP(&dbDir, "path", "p", "./", "Dir path to DB")
	dumpCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Print everything")
	dumpCmd.Flags().IntVarP(&cacheSize, "cache", "c", 10000, "Size of the Cache")
}

func DumpDatabase(cmd *cobra.Command, args []string) {
	if dbName == "" {
		dbName = "merkleeyes"
	}

	dbPath := path.Join(dbDir, dbName+".db")

	if !cmn.FileExists(dbPath) {
		cmn.Exit("No existing database: " + dbPath)
	}

	if verbose {
		fmt.Printf("Dumping DB %s (%s)...\n", dbName, dbType)
	}

	database := db.NewDB(dbName, db.LevelDBBackendStr, "./")

	if verbose {
		fmt.Printf("Database: %v\n", database)
	}

	tree := iavl.NewIAVLTree(cacheSize, database)

	if verbose {
		fmt.Printf("Tree: %v\n", tree)
	}

	tree.Dump(verbose, nil)
}
