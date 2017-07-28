package cmd

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/tendermint/merkleeyes/iavl"
	"github.com/tendermint/tmlibs/db"
	"github.com/tendermint/tmlibs/merkle"
)

var loadtestCmd = &cobra.Command{
	Run:   LoadTest,
	Use:   "loadtest",
	Short: "Run a load test on the database",
	Long:  `Do a long running load test on the database to determine timing`,
}

const reportInterval = 100

var (
	initSize  int
	keySize   int
	dataSize  int
	blockSize int
)

func init() {
	RootCmd.AddCommand(loadtestCmd)
	loadtestCmd.Flags().IntVarP(&initSize, "initsize", "i", 100000, "Initial DB Size")
	loadtestCmd.Flags().IntVarP(&keySize, "keysize", "k", 16, "Length of keys (in bytes)")
	loadtestCmd.Flags().IntVarP(&dataSize, "valuesize", "v", 100, "Length of values (in bytes)")
	loadtestCmd.Flags().IntVarP(&blockSize, "blocksize", "b", 200, "Number of Txs per block")
}

func LoadTest(cmd *cobra.Command, args []string) {

	tmpDir, err := ioutil.TempDir("", "loadtest-")
	if err != nil {
		fmt.Printf("Cannot create temp dir: %s\n", err)
		os.Exit(-1)
	}

	initMB := memUseMB()
	start := time.Now()

	fmt.Printf("Preparing DB (%s with %d keys)...\n", dbType, initSize)
	d := db.NewDB("loadtest", dbType, tmpDir)
	tree, keys := prepareTree(d, initSize, keySize, dataSize)

	delta := time.Now().Sub(start)
	fmt.Printf("Initialization took %0.3f s, used %0.2f MB\n",
		delta.Seconds(), memUseMB()-initMB)
	fmt.Printf("Keysize: %d, Datasize: %d\n", keySize, dataSize)

	fmt.Printf("Starting loadtest (blocks of %d tx)...\n", blockSize)
	loopForever(tree, dataSize, blockSize, keys, initMB)
}

// blatently copied from benchmarks/bench_test.go
func randBytes(length int) []byte {
	key := make([]byte, length)
	// math.rand.Read always returns err=nil
	rand.Read(key)
	return key
}

// blatently copied from benchmarks/bench_test.go
func prepareTree(db db.DB, size, keyLen, dataLen int) (merkle.Tree, [][]byte) {
	t := iavl.NewIAVLTree(size, db)
	keys := make([][]byte, size)

	for i := 0; i < size; i++ {
		key := randBytes(keyLen)
		t.Set(key, randBytes(dataLen))
		keys[i] = key
	}
	t.Hash()
	t.Save()
	runtime.GC()
	return t, keys
}

func runBlock(t merkle.Tree, dataLen, blockSize int, keys [][]byte) merkle.Tree {
	l := int32(len(keys))

	real := t.Copy()
	check := t.Copy()

	for j := 0; j < blockSize; j++ {
		// always update to avoid changing size
		key := keys[rand.Int31n(l)]
		data := randBytes(dataLen)

		// perform query and write on check and then real
		check.Get(key)
		check.Set(key, data)
		real.Get(key)
		real.Set(key, data)
	}

	// at the end of a block, move it all along....
	real.Hash()
	real.Save()
	return real
}

func loopForever(t merkle.Tree, dataLen, blockSize int, keys [][]byte, initMB float64) {
	for {
		start := time.Now()
		for i := 0; i < reportInterval; i++ {
			t = runBlock(t, dataLen, blockSize, keys)
		}
		// now report
		end := time.Now()
		delta := end.Sub(start)
		timing := delta.Seconds() / reportInterval
		usedMB := memUseMB() - initMB
		fmt.Printf("%s: blocks of %d tx: %0.3f s/block, using %0.2f MB\n",
			end.Format("Jan 2 15:04:05"), blockSize, timing, usedMB)
	}
}

// returns number of MB in use
func memUseMB() float64 {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	asize := mem.Alloc
	mb := float64(asize) / 1000000
	return mb
}
