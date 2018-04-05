package types

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/encoding/blockchain"
	"github.com/bytom/testutil"
)

func TestBlockHeader(t *testing.T) {
	blockHeader := &BlockHeader{
		Version:           1,
		Height:            432234,
		PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
		Timestamp:         1522908275,
		Nonce:             34342,
		Bits:              2305843009222082559,
		BlockCommitment: BlockCommitment{
			TransactionStatusHash:  testutil.MustDecodeHash("b94301ea4e316bee00109f68d25beaca90aeff08e9bf439a37d91d7a3b5a1470"),
			TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
		},
	}

	wantHex := strings.Join([]string{
		"01",     // serialization flags
		"01",     // version
		"eab01a", // block height
		"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
		"f3f896d605", // timestamp
		"40",         // commitment extensible field length
		"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
		"b94301ea4e316bee00109f68d25beaca90aeff08e9bf439a37d91d7a3b5a1470", // tx status hash
		"a68c02",             // nonce
		"ffffff838080808020", // bits
	}, "")

	gotHex := testutil.Serialize(t, blockHeader)
	want, err := hex.DecodeString(wantHex)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(gotHex, want) {
		t.Errorf("empty block header bytes = %x want %x", gotHex, want)
	}

	gotBlockHeader := BlockHeader{}
	if _, err := gotBlockHeader.readFrom(blockchain.NewReader(want)); err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(gotBlockHeader, *blockHeader) {
		t.Errorf("got:\n%s\nwant:\n%s", spew.Sdump(gotBlockHeader), spew.Sdump(*blockHeader))
	}
}
