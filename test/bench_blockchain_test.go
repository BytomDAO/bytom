package test

import (
	"io/ioutil"
	"os"
	"testing"
)

func BenchmarkChain_CoinBaseTx_NoAsset(b *testing.B) {
	benchInsertChain(b, 0, 0, "")
}

func BenchmarkChain_BtmTx_NoAsset_BASE(b *testing.B) {
	benchInsertChain(b, 1, 0, "")
}

func BenchmarkChain_5000BtmTx_NoAsset_BASE(b *testing.B) {
	benchInsertChain(b, 5000, 0, "")
}

func BenchmarkChain_5000BtmTx_5Asset_BASE(b *testing.B) {
	benchInsertChain(b, 5000, 5, "")
}

func BenchmarkChain_10000BtmTx_NoAsset_BASE(b *testing.B) {
	benchInsertChain(b, 10000, 0, "")
}

func BenchmarkChain_10000BtmTx_1Asset_BASE(b *testing.B) {
	benchInsertChain(b, 10000, 1, "")
}

func BenchmarkChain_10000BtmTx_5Asset_BASE(b *testing.B) {
	benchInsertChain(b, 10000, 5, "")
}

// standard Transaction
func BenchmarkChain_BtmTx_NoAsset_P2PKH(b *testing.B) {
	benchInsertChain(b, 5000, 0, "P2PKH")
}

func BenchmarkChain_BtmTx_1Asset_P2PKH(b *testing.B) {
	benchInsertChain(b, 5000, 1, "P2PKH")
}

func BenchmarkChain_BtmTx_NoAsset_P2SH(b *testing.B) {
	benchInsertChain(b, 3000, 0, "P2SH")
}

func BenchmarkChain_BtmTx_1Asset_P2SH(b *testing.B) {
	benchInsertChain(b, 3000, 1, "P2SH")
}

func BenchmarkChain_BtmTx_NoAsset_MutiSign(b *testing.B) {
	benchInsertChain(b, 3000, 0, "MutiSign")
}

func BenchmarkChain_BtmTx_1Asset_MutiSign(b *testing.B) {
	benchInsertChain(b, 3000, 1, "MutiSign")
}

func benchInsertChain(b *testing.B, blockTxNumber int, otherAssetNum int, txType string) {
	b.StopTimer()
	testNumber := b.N
	totalTxNumber := testNumber * blockTxNumber

	dirPath, err := ioutil.TempDir(".", "testDB")
	if err != nil {
		b.Fatal("create dirPath err:", err)
	}
	defer os.RemoveAll(dirPath)

	// Generate a chain test data.
	chain, txs, txPool, err := GenerateChainData(dirPath, totalTxNumber, otherAssetNum, txType)
	if err != nil {
		b.Fatal("GenerateChainData err:", err)
	}

	b.ReportAllocs()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		testTxs := txs[blockTxNumber*i : blockTxNumber*(i+1)]
		if err := InsertChain(chain, txPool, testTxs); err != nil {
			b.Fatal("Failed to insert block into chain:", err)
		}
	}
}
