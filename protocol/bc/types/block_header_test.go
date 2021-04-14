package types

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/testutil"
)

func TestBlockHeader(t *testing.T) {
	blockHeader := &BlockHeader{
		Version:           1,
		Height:            432234,
		PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
		Timestamp:         1522908275,
		BlockCommitment: BlockCommitment{
			TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
		},
	}

	wantHex := strings.Join([]string{
		"01",     // serialization flags
		"01",     // version
		"eab01a", // block height
		"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
		"f3f896d605", // timestamp
		"20",         // commitment extensible field length
		"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
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

func TestMarshalBlockHeader(t *testing.T) {
	cases := []struct {
		blockHeader *BlockHeader
		wantHex     string
		wantError   error
	}{
		{
			blockHeader: &BlockHeader{
				Version:           1,
				Height:            10000,
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         1528945000,
				BlockCommitment: BlockCommitment{
					TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
				},
			},
			wantHex: strings.Join([]string{
				"01",   // serialization flags
				"01",   // version
				"904e", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"20",         // commitment extensible field length
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
			}, ""),
		},
		{
			blockHeader: &BlockHeader{
				Version:           1,
				Height:            9223372036854775808, // Height > MaxInt64(9223372036854775807)
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         1528945000,
				BlockCommitment: BlockCommitment{
					TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
				},
			},
			wantError: blockchain.ErrRange,
		},
		{
			blockHeader: &BlockHeader{
				Version:           1,
				Height:            10000,
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         9223372036854775808, // Timestamp > MaxInt64(9223372036854775807)
				BlockCommitment: BlockCommitment{
					TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
				},
			},
			wantError: blockchain.ErrRange,
		},
		{
			blockHeader: &BlockHeader{
				Version:           1,
				Height:            9223372036854775807, // MaxInt64(9223372036854775807)
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         1528945000,
			},
			wantHex: strings.Join([]string{
				"01",                 // serialization flags
				"01",                 // version
				"ffffffffffffffff7f", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"20",         // commitment extensible field length
				"0000000000000000000000000000000000000000000000000000000000000000", // transactions merkle root
			}, ""),
		},
	}

	for i, test := range cases {
		got, err := test.blockHeader.MarshalText()
		if err != nil && err != test.wantError {
			t.Errorf("test %d: got error = %x, want = %x", i, err, test.wantError)
		} else if err != nil && err == test.wantError {
			continue
		}

		if string(got) != test.wantHex {
			t.Errorf("test %d: got strbytes = %s, want %s", i, string(got), test.wantHex)
		}

		resultBlockHeader := &BlockHeader{}
		if err := resultBlockHeader.UnmarshalText(got); err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(*resultBlockHeader, *test.blockHeader) {
			t.Errorf("test %d: got:\n%s\nwant:\n%s", i, spew.Sdump(*resultBlockHeader), spew.Sdump(*test.blockHeader))
		}
	}
}

func TestUnmarshalBlockHeader(t *testing.T) {
	cases := []struct {
		hexBlockHeader  string
		wantBlockHeader *BlockHeader
		wantError       error
	}{
		{
			hexBlockHeader: strings.Join([]string{
				"01",   // serialization flags (SerBlockHeader = 01)
				"01",   // version
				"904e", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"20",         // commitment extensible field length
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
			}, ""),
			wantBlockHeader: &BlockHeader{
				Version:           1,
				Height:            10000,
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         1528945000,
				BlockCommitment: BlockCommitment{
					TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
				},
			},
		},
		{
			hexBlockHeader: strings.Join([]string{
				"03",   // serialization flags (SerBlockFull = 03)
				"01",   // version
				"904e", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"20",         // commitment extensible field length
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
			}, ""),
			wantBlockHeader: &BlockHeader{
				Version:           1,
				Height:            10000,
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         1528945000,
				BlockCommitment: BlockCommitment{
					TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
				},
			},
		},
		{
			hexBlockHeader: strings.Join([]string{
				"02",   // serialization flags (SerBlockTransactions = 02)
				"01",   // version
				"904e", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"20",         // commitment extensible field length
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
			}, ""),
			wantError: fmt.Errorf("unsupported serialization flags 0x02"),
		},
		{
			hexBlockHeader: strings.Join([]string{
				"01",  // serialization flags
				"01",  // version
				"908", // block height (error with odd length)
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"20",         // commitment extensible field length
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
			}, ""),
			wantError: hex.ErrLength,
		},
		{
			hexBlockHeader: strings.Join([]string{
				"01",                 // serialization flags
				"01",                 // version
				"ffffffffffffffffff", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"20",         // commitment extensible field length
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
			}, ""),
			wantError: errors.New("binary: varint overflows a 64-bit integer"),
		},
		{
			hexBlockHeader: strings.Join([]string{
				"01",                 // serialization flags
				"01",                 // version
				"ffffffffffffffff7f", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"20",         // commitment extensible field length
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
			}, ""),
			wantBlockHeader: &BlockHeader{
				Version:           1,
				Height:            9223372036854775807, // MaxInt64(9223372036854775807)
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         1528945000,
				BlockCommitment: BlockCommitment{
					TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
				},
			},
		},
	}

	for i, test := range cases {
		resultBlockHeader := &BlockHeader{}
		err := resultBlockHeader.UnmarshalText([]byte(test.hexBlockHeader))
		if err != nil && err.Error() != test.wantError.Error() {
			t.Errorf("test %d: got error = %s, want = %s", i, err.Error(), test.wantError.Error())
		} else if err != nil && err.Error() == test.wantError.Error() {
			continue
		}

		if !testutil.DeepEqual(*resultBlockHeader, *test.wantBlockHeader) {
			t.Errorf("test %d: got:\n%s\nwant:\n%s", i, spew.Sdump(*resultBlockHeader), spew.Sdump(*test.wantBlockHeader))
		}
	}
}
