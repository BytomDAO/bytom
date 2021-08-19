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
		BlockWitness: testutil.MustDecodeHexString("3b71527c26bff856a0f247df4fce1b48780f1bd9fd55ba79fb637b0f2e897bb019c5449febf593032dd25b9027cea712c752104700e67d8813326b06d052bf00"),
	}

	wantHex := strings.Join([]string{
		"01",     // serialization flags
		"01",     // version
		"eab01a", // block height
		"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
		"f3f896d605", // timestamp
		"20",         // commitment extensible field length
		"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03",                                                                     // transactions merkle root
		"41403b71527c26bff856a0f247df4fce1b48780f1bd9fd55ba79fb637b0f2e897bb019c5449febf593032dd25b9027cea712c752104700e67d8813326b06d052bf00", // block witness
		"0100", // supLinks
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
				BlockWitness: testutil.MustDecodeHexString("9dc8892df991e1d1110a5cb1bbfd57f2f5e3aa89464de50f9555c7575d9c2b21cf8f05b77b880d8ae4dd218efb15b775c32c9d77f9a2955d69dca9020c127709"),
			},
			wantHex: strings.Join([]string{
				"01",   // serialization flags
				"01",   // version
				"904e", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"20",         // commitment extensible field length
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03",                                                                     // transactions merkle root
				"41409dc8892df991e1d1110a5cb1bbfd57f2f5e3aa89464de50f9555c7575d9c2b21cf8f05b77b880d8ae4dd218efb15b775c32c9d77f9a2955d69dca9020c127709", // block witness
				"0100", // supLinks
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
				BlockWitness: testutil.MustDecodeHexString("2c27ea6e848a1191f25a7f4a04deae1c5a191587e5ee61f92e408ab97dbd35c3ce613b08475f0baa300606c38695d1eb0c4b409939acaa28b82fbb87e7de3c0f"),
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
				BlockWitness: testutil.MustDecodeHexString("3dc3cc903772033972b9f3e48df330649c8e6beb3a4376e694b83dedb91da8692a32da3817edf1606cd5800a411f91316c96b2700a275d22c52d5fdc28e0fa03"),
			},
			wantError: blockchain.ErrRange,
		},
		{
			blockHeader: &BlockHeader{
				Version:           1,
				Height:            9223372036854775807, // MaxInt64(9223372036854775807)
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         1528945000,
				BlockWitness:      testutil.MustDecodeHexString("ac6380a17685c48af4b0a0877d9d61e83b50bd95daa61083dd90031ae66d12d7a371c41cce24887d4d422202b747494bb0e7ca78567d6663be82b27714357407"),
			},
			wantHex: strings.Join([]string{
				"01",                 // serialization flags
				"01",                 // version
				"ffffffffffffffff7f", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"20",         // commitment extensible field length
				"0000000000000000000000000000000000000000000000000000000000000000",                                                                     // transactions merkle root
				"4140ac6380a17685c48af4b0a0877d9d61e83b50bd95daa61083dd90031ae66d12d7a371c41cce24887d4d422202b747494bb0e7ca78567d6663be82b27714357407", // block witness
				"0100", // supLinks
			}, ""),
		},
		{
			blockHeader: &BlockHeader{
				Version:           1,
				Height:            9223372036854775807, // MaxInt64(9223372036854775807)
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         1528945000,
				BlockWitness:      BlockWitness{},
			},
			wantHex: strings.Join([]string{
				"01",                 // serialization flags
				"01",                 // version
				"ffffffffffffffff7f", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"20",         // commitment extensible field length
				"0000000000000000000000000000000000000000000000000000000000000000", // transactions merkle root
				"0100", // block witness
				"0100", // supLinks
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
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03",                                                                     // transactions merkle root
				"41409dc8892df991e1d1110a5cb1bbfd57f2f5e3aa89464de50f9555c7575d9c2b21cf8f05b77b880d8ae4dd218efb15b775c32c9d77f9a2955d69dca9020c127709", // block witness
				"0100", // supLinks
			}, ""),
			wantBlockHeader: &BlockHeader{
				Version:           1,
				Height:            10000,
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         1528945000,
				BlockCommitment: BlockCommitment{
					TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
				},
				BlockWitness: testutil.MustDecodeHexString("9dc8892df991e1d1110a5cb1bbfd57f2f5e3aa89464de50f9555c7575d9c2b21cf8f05b77b880d8ae4dd218efb15b775c32c9d77f9a2955d69dca9020c127709"),
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
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03",                                                                     // transactions merkle root
				"41409dc8892df991e1d1110a5cb1bbfd57f2f5e3aa89464de50f9555c7575d9c2b21cf8f05b77b880d8ae4dd218efb15b775c32c9d77f9a2955d69dca9020c127709", // block witness
				"0100", // supLinks
			}, ""),
			wantBlockHeader: &BlockHeader{
				Version:           1,
				Height:            10000,
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         1528945000,
				BlockCommitment: BlockCommitment{
					TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
				},
				BlockWitness: testutil.MustDecodeHexString("9dc8892df991e1d1110a5cb1bbfd57f2f5e3aa89464de50f9555c7575d9c2b21cf8f05b77b880d8ae4dd218efb15b775c32c9d77f9a2955d69dca9020c127709"),
			},
		},
		{
			hexBlockHeader: strings.Join([]string{
				"02",   // serialization flags (SerBlockTransactions = 02)
				"01",   // version
				"904e", // block height
				"c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0", // prev block hash
				"e8b287d905", // timestamp
				"0100",       // BlockWitness
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
				"0100",       // BlockWitness
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
				"0100",       // BlockWitness
				"20",         // commitment extensible field length
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03", // transactions merkle root
				"0100", // block witness
				"0100", // supLinks
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
				"ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03",                                                                     // transactions merkle root
				"4140e0776a3cf17b3e0f8340caeee32a75d02ecc25cf20bee9e5c7503bca3b2703c3c61fdcb4211ed59b58eb025ac81e06b138d54b5d01ea4614dd0f65e641836900", // block witness
				"0100", // supLinks
			}, ""),
			wantBlockHeader: &BlockHeader{
				Version:           1,
				Height:            9223372036854775807, // MaxInt64(9223372036854775807)
				PreviousBlockHash: testutil.MustDecodeHash("c34048bd60c4c13144fd34f408627d1be68f6cb4fdd34e879d6d791060ea73a0"),
				Timestamp:         1528945000,
				BlockCommitment: BlockCommitment{
					TransactionsMerkleRoot: testutil.MustDecodeHash("ad9ac003d08ff305181a345d64fe0b02311cc1a6ec04ab73f3318d90139bfe03"),
				},
				BlockWitness: testutil.MustDecodeHexString("e0776a3cf17b3e0f8340caeee32a75d02ecc25cf20bee9e5c7503bca3b2703c3c61fdcb4211ed59b58eb025ac81e06b138d54b5d01ea4614dd0f65e641836900"),
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
