package state

import (
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/database/storage"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
)

func coinBaseTx(amount uint64) *types.Tx {
	return types.NewTx(types.TxData{
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput([]byte("arbitrary")),
		},
		Outputs: []*types.TxOutput{
			types.NewTxOutput(*consensus.BTMAssetID, amount, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
		},
	})
}

func spendTx(strHash string, amount, sourcePos uint64) *types.Tx {
	sourceID := bc.Hash{}
	sourceID.UnmarshalText([]byte(strHash))
	return types.NewTx(types.TxData{
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, sourceID, *consensus.BTMAssetID, amount, sourcePos, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
		},
		Outputs: []*types.TxOutput{
			types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("00148c704747e94387fa0b8712b053ed2132d84820ac")),
			types.NewTxOutput(*consensus.BTMAssetID, amount-100000000, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
		},
	})
}

var testBlocks = []*bc.Block{
	&bc.Block{
		BlockHeader: &bc.BlockHeader{
			Height:            100,
			PreviousBlockId:   &bc.Hash{V0: 770850060912030643, V1: 13252854004788846472, V2: 3713302034413986133, V3: 11125523136536745575},
			Timestamp:         1522908275,
			Nonce:             0,
			TransactionStatus: bc.NewTransactionStatus(),
		},
		Transactions: []*bc.Tx{
			coinBaseTx(41250000000).Tx,
		},
	},
	&bc.Block{
		BlockHeader: &bc.BlockHeader{
			Height:            201,
			PreviousBlockId:   &bc.Hash{V0: 770850060912030643, V1: 13252854004788846472, V2: 3713302034413986133, V3: 11125523136536745575},
			Timestamp:         1522908275,
			Nonce:             0,
			TransactionStatus: bc.NewTransactionStatus(),
		},
		Transactions: []*bc.Tx{
			coinBaseTx(41350000000).Tx,
			spendTx("9e74e35362ffc73c8967aa0008da8fcbc62a21d35673fb970445b5c2972f8603", 41250000000, 0).Tx,
			spendTx("3e79a6d70596daf6ebe0c16574b7388def9049e77de868874cc26f480edafde1", 41150000000, 1).Tx,
			spendTx("7666c17e82cbef8ab756197253d3781d5993f129521adff244447ff73e03ac42", 41050000000, 1).Tx,
		},
	},
	&bc.Block{
		BlockHeader: &bc.BlockHeader{
			Height:            202,
			PreviousBlockId:   &bc.Hash{V0: 770850060912030643, V1: 13252854004788846472, V2: 3713302034413986133, V3: 11125523136536745575},
			Timestamp:         1522908275,
			Nonce:             0,
			TransactionStatus: bc.NewTransactionStatus(),
		},
		Transactions: []*bc.Tx{
			coinBaseTx(41350000000).Tx,
			spendTx("6b231b711c9533c92d5ddd8ec26aac4903668d689da94cb924c5cd3323924c9b", 40950000000, 1).Tx,
		},
	},
	&bc.Block{
		BlockHeader: &bc.BlockHeader{
			Height:            203,
			PreviousBlockId:   &bc.Hash{V0: 770850060912030643, V1: 13252854004788846472, V2: 3713302034413986133, V3: 11125523136536745575},
			Timestamp:         1522908275,
			Nonce:             0,
			TransactionStatus: bc.NewTransactionStatus(),
		},
		Transactions: []*bc.Tx{
			coinBaseTx(41350000000).Tx,
			spendTx("50ee363bd24c7ad38efca1480cf740de4b505d884bcb70e647f9b4c12803592f", 40850000000, 1).Tx,
			spendTx("ddd2a083b0e6739ed8fad7d40296e03709cd22c6c76a245adf36514139fd209e", 40750000000, 1).Tx,
		},
	},
	&bc.Block{
		BlockHeader: &bc.BlockHeader{
			Height:            204,
			PreviousBlockId:   &bc.Hash{V0: 770850060912030643, V1: 13252854004788846472, V2: 3713302034413986133, V3: 11125523136536745575},
			Timestamp:         1522908275,
			Nonce:             0,
			TransactionStatus: bc.NewTransactionStatus(),
		},
		Transactions: []*bc.Tx{
			coinBaseTx(41350000000).Tx,
			spendTx("efa48a0bda2c6b1db5eb14b497be0bd1a18b0b62337050d8634dca3d5a1ef504", 40650000000, 1).Tx,
		},
	},
}

func blockNode(header bc.BlockHeader) *state.BlockNode {
	h := types.BlockHeader{
		Version:           header.Version,
		Height:            header.Height,
		PreviousBlockHash: *header.PreviousBlockId,
		Timestamp:         header.Timestamp,
		Bits:              header.Bits,
		Nonce:             header.Nonce,
	}
	return &state.BlockNode{
		Parent:    nil,
		Hash:      h.Hash(),
		WorkSum:   difficulty.CalcWork(h.Bits),
		Version:   h.Version,
		Height:    h.Height,
		Timestamp: h.Timestamp,
		Nonce:     h.Nonce,
		Bits:      h.Bits,
	}
}

const utxoPreFix = "UT:"

func calcUtxoKey(hash *bc.Hash) []byte {
	return []byte(utxoPreFix + hash.String())
}

func TestAttacOrDetachBlocks(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	store := leveldb.NewStore(testDB)

	cases := []struct {
		desc     string
		before   map[string]*storage.UtxoEntry
		want     map[string]*storage.UtxoEntry
		block    *bc.Block
		txStatus *bc.TransactionStatus
	}{
		{
			desc:   "coinbase tx",
			before: make(map[string]*storage.UtxoEntry),
			want:   map[string]*storage.UtxoEntry{"b3e02c46ae716fab82b63ea3c56ed9405664a2947dfe79087e81fc516de86bfb": storage.NewUtxoEntry(true, 100, false)},
			block:  testBlocks[0],
			txStatus: &bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
				&bc.TxVerifyResult{StatusFail: true},
			}},
		},
		{
			desc:   "Chain trading 3",
			before: map[string]*storage.UtxoEntry{"b3e02c46ae716fab82b63ea3c56ed9405664a2947dfe79087e81fc516de86bfb": storage.NewUtxoEntry(true, 100, false)},
			want: map[string]*storage.UtxoEntry{
				"b3e02c46ae716fab82b63ea3c56ed9405664a2947dfe79087e81fc516de86bfb": storage.NewUtxoEntry(true, 201, true),
				"a5b92ceea7c36439abe1ae187f27e7a31d1835cbb42d90d5f837a98d5be4cd89": storage.NewUtxoEntry(false, 201, true),
				"465795964bee6851fd8bb6e600f3428d29f6c4c6158c08812a6b068ea00cda8e": storage.NewUtxoEntry(false, 201, true),
				"10543a888f2c4e3691533e5f769f729ab8937096f5951f17688db3a070d35d88": storage.NewUtxoEntry(false, 201, false),
			},
			block: testBlocks[1],
			txStatus: &bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
				&bc.TxVerifyResult{StatusFail: true},
				&bc.TxVerifyResult{StatusFail: true},
				&bc.TxVerifyResult{StatusFail: true},
				&bc.TxVerifyResult{StatusFail: true},
			}},
		},
		{
			desc:   "single tx",
			before: map[string]*storage.UtxoEntry{"10543a888f2c4e3691533e5f769f729ab8937096f5951f17688db3a070d35d88": storage.NewUtxoEntry(false, 201, false)},
			want: map[string]*storage.UtxoEntry{
				"10543a888f2c4e3691533e5f769f729ab8937096f5951f17688db3a070d35d88": storage.NewUtxoEntry(false, 202, true),
				"a8f85e3497d910cacd50f8930ee5bd06537dae0b61f87b85d760e19ae27a69e8": storage.NewUtxoEntry(false, 202, false),
			},
			block: testBlocks[2],
			txStatus: &bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
				&bc.TxVerifyResult{StatusFail: true},
				&bc.TxVerifyResult{StatusFail: true},
			}},
		},
		{
			desc:   "The rollback block 2",
			before: map[string]*storage.UtxoEntry{"a8f85e3497d910cacd50f8930ee5bd06537dae0b61f87b85d760e19ae27a69e8": storage.NewUtxoEntry(false, 202, false)},
			want: map[string]*storage.UtxoEntry{
				"a8f85e3497d910cacd50f8930ee5bd06537dae0b61f87b85d760e19ae27a69e8": storage.NewUtxoEntry(false, 203, true),
				"c92e3f22618e4eea9b756bd84b0e079e3cafcf7d3d5c6cec949e500037910e32": storage.NewUtxoEntry(false, 203, true),
				"ca9b179e549406aa583869e124e39817414d4500a8ce5476e95b6018d182b966": storage.NewUtxoEntry(false, 203, false),
			},
			block: testBlocks[3],
			txStatus: &bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
				&bc.TxVerifyResult{StatusFail: true},
				&bc.TxVerifyResult{StatusFail: true},
				&bc.TxVerifyResult{StatusFail: true},
			}},
		},

		{
			desc:   "The rollback block",
			before: map[string]*storage.UtxoEntry{"ca9b179e549406aa583869e124e39817414d4500a8ce5476e95b6018d182b966": storage.NewUtxoEntry(false, 203, false)},
			want: map[string]*storage.UtxoEntry{
				"ca9b179e549406aa583869e124e39817414d4500a8ce5476e95b6018d182b966": storage.NewUtxoEntry(false, 203, true),
			},
			block: testBlocks[4],
			txStatus: &bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
				&bc.TxVerifyResult{StatusFail: true},
				&bc.TxVerifyResult{StatusFail: true},
			}},
		},
	}
	utxoViewpoint := state.NewUtxoViewpoint()
	for _, c := range cases {
		store.GetTransactionsUtxo(utxoViewpoint, c.block.Transactions)
		utxoViewpoint.ApplyBlock(c.block, c.txStatus)
		node := blockNode(*c.block.BlockHeader)
		store.SaveChainStatus(node, utxoViewpoint)

		for k, v := range c.want {
			// Comparison before data
			if d, ok := c.before[k]; ok {
				if v.Spent == d.Spent {
					t.Errorf("Same data before and after: %s:%v", k, v)
				}
			}

			h := bc.Hash{}
			h.UnmarshalText([]byte(k))

			if u, ok := utxoViewpoint.Entries[h]; ok {
				if u.Spent != u.Spent {
					t.Errorf("The result data is incorrect: %s:%v", k, v)
				}
			}

			if data := testDB.Get(calcUtxoKey(&h)); data != nil {
				utxoEntry := storage.UtxoEntry{}
				if err := proto.Unmarshal(data, &utxoEntry); err != nil {
					t.Error(err)
				}

				if v.Spent != utxoEntry.Spent {
					t.Errorf("The utxoEntry data in the database is incorrect: %s:%v", k, v)
				}
			}

		}

	}
}
