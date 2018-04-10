package test

import (
	"testing"
)

func BenchmarkInsertChain_CoinBaseTx_NoOtherAsset(b *testing.B) {
	benchInsertChain(b, 0, 0, "")
}

func BenchmarkInsertChain_BtmTx_NoOtherAsset(b *testing.B) {
	benchInsertChain(b, 1, 0, "")
}

func BenchmarkInsertChain_10BtmTx_NoOtherAsset(b *testing.B) {
	benchInsertChain(b, 10, 0, "")
}

func BenchmarkInsertChain_200BtmTx_NoOtherAsset(b *testing.B) {
	benchInsertChain(b, 200, 0, "")
}

func BenchmarkInsertChain_3000BtmTx_NoOtherAsset(b *testing.B) {
	benchInsertChain(b, 3000, 0,"")
}

func BenchmarkInsertChain_BtmTx_OtherAsset(b *testing.B) {
	benchInsertChain(b, 1, 1,"")
}

func BenchmarkInsertChain_200BtmTx_OtherAsset(b *testing.B) {
	benchInsertChain(b, 200, 10,"")
}

func BenchmarkInsertChain_1000BtmTx_OtherAsset(b *testing.B) {
	benchInsertChain(b, 1000, 10, "")
}

func BenchmarkInsertChain_BtmTx_NoOtherAsset_P2PKH(b *testing.B) {
	benchInsertChain(b, 10, 0, "P2PKH")
}

func BenchmarkInsertChain_BtmTx_NoOtherAsset_P2SH(b *testing.B) {
	benchInsertChain(b, 10, 0, "P2SH")
}

func BenchmarkInsertChain_BtmTx_NoOtherAsset_MutiSign(b *testing.B) {
	benchInsertChain(b, 10, 0, "MutiSign")
}

func benchInsertChain(b *testing.B, blockTxNumber int, otherAssetNum int, txType string) {
	testNumber := b.N
	totalTxNumber := testNumber * blockTxNumber

	// Generate a chain test data.
	chain, txs, err := GenerateChainData(totalTxNumber, otherAssetNum, txType)
	if err != nil {
		b.Fatal("GenerateChainData err:", err)
	}

	// Set the time for inserting block into the new chain.
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < testNumber; i++ {
		testTxs := txs[blockTxNumber*i : blockTxNumber*(i+1)]
		if err := InsertChain(chain, testTxs); err != nil {
			b.Fatal("Failed to insert block into chain:", err)
		}
	}
}
