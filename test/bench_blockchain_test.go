package test

import (
	"testing"
)

func BenchmarkInsertChain_CoinBaseTx_NoAsset(b *testing.B) {
	benchInsertChain(b, 0, 0, "")
}

func BenchmarkChain_BtmTx_NoAsset(b *testing.B) {
	benchInsertChain(b, 1, 0, "")
}

func BenchmarkChain_10BtmTx_NoAsset(b *testing.B) {
	benchInsertChain(b, 10, 0, "")
}

func BenchmarkChain_200BtmTx_NoAsset(b *testing.B) {
	benchInsertChain(b, 200, 0, "")
}

func BenchmarkChain_3000BtmTx_NoAsset(b *testing.B) {
	benchInsertChain(b, 3000, 0,"")
}

func BenchmarkChain_BtmTx_1Asset(b *testing.B) {
	benchInsertChain(b, 1, 1,"")
}

func BenchmarkChain_200BtmTx_10Asset(b *testing.B) {
	benchInsertChain(b, 200, 10,"")
}

func BenchmarkChain_1000BtmTx_10Asset(b *testing.B) {
	benchInsertChain(b, 1000, 10, "")
}

// standard Transaction
func BenchmarkChain_BtmTx_NoAsset_P2PKH(b *testing.B) {
	benchInsertChain(b, 1000, 0, "P2PKH")
}

func BenchmarkChain_tmTx_10Asset_P2PKH(b *testing.B) {
	benchInsertChain(b, 300, 100, "P2PKH")
}

func BenchmarkChain_BtmTx_NoAsset_P2SH(b *testing.B) {
	benchInsertChain(b, 1000, 0, "P2SH")
}

func BenchmarkChain_BtmTx_10Asset_P2SH(b *testing.B) {
	benchInsertChain(b, 1000, 10, "P2SH")
}

func BenchmarkChain_BtmTx_NoAsset_MutiSign(b *testing.B) {
	benchInsertChain(b, 1000, 0, "MutiSign")
}

func BenchmarkChain_BtmTx_10Asset_MutiSign(b *testing.B) {
	benchInsertChain(b, 1000, 10, "MutiSign")
}

func benchInsertChain(b *testing.B, blockTxNumber int, otherAssetNum int, txType string) {
	testNumber := b.N
	totalTxNumber := testNumber * blockTxNumber

	// Generate a chain test data.
	chain, txs, txPool, err := GenerateChainData(totalTxNumber, otherAssetNum, txType)
	if err != nil {
		b.Fatal("GenerateChainData err:", err)
	}

	// Set the time for inserting block into the new chain.
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < testNumber; i++ {
		testTxs := txs[blockTxNumber*i : blockTxNumber*(i+1)]
		if err := InsertChain(chain, txPool, testTxs); err != nil {
			b.Fatal("Failed to insert block into chain:", err)
		}
	}
}
