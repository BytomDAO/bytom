package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/bytom/bytom/account"
	"github.com/bytom/bytom/consensus"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/testutil"
)

func TestGetAccountUtxos(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	cases := []struct {
		dbUtxos          map[string]*account.UTXO
		unconfirmedUtxos []*account.UTXO
		id               string
		unconfirmed      bool
		isSmartContract  bool
		wantUtxos        []*account.UTXO
	}{
		{
			dbUtxos:         map[string]*account.UTXO{},
			id:              "",
			unconfirmed:     false,
			isSmartContract: false,
			wantUtxos:       []*account.UTXO{},
		},
		{
			dbUtxos: map[string]*account.UTXO{
				string(account.StandardUTXOKey(bc.Hash{V0: 1})): &account.UTXO{
					OutputID: bc.Hash{V0: 1},
				},
				string(account.StandardUTXOKey(bc.Hash{V0: 2})): &account.UTXO{
					OutputID: bc.Hash{V0: 2},
				},
				string(account.StandardUTXOKey(bc.Hash{V0: 3})): &account.UTXO{
					OutputID: bc.Hash{V0: 3},
				},
				string(account.ContractUTXOKey(bc.Hash{V0: 4})): &account.UTXO{
					OutputID: bc.Hash{V0: 4},
				},
			},
			unconfirmedUtxos: []*account.UTXO{},
			id:               "",
			isSmartContract:  false,
			wantUtxos: []*account.UTXO{
				&account.UTXO{OutputID: bc.Hash{V0: 1}},
				&account.UTXO{OutputID: bc.Hash{V0: 2}},
				&account.UTXO{OutputID: bc.Hash{V0: 3}},
			},
		},
		{
			dbUtxos: map[string]*account.UTXO{
				string(account.StandardUTXOKey(bc.Hash{V0: 1})): &account.UTXO{
					OutputID: bc.Hash{V0: 1},
				},
				string(account.StandardUTXOKey(bc.Hash{V0: 2})): &account.UTXO{
					OutputID: bc.Hash{V0: 2},
				},
				string(account.StandardUTXOKey(bc.Hash{V0: 3})): &account.UTXO{
					OutputID: bc.Hash{V0: 3},
				},
				string(account.ContractUTXOKey(bc.Hash{V0: 4})): &account.UTXO{
					OutputID: bc.Hash{V0: 4},
				},
			},
			unconfirmedUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 5},
					ControlProgram: []byte("smart contract"),
				},
			},
			id:              "",
			unconfirmed:     false,
			isSmartContract: true,
			wantUtxos: []*account.UTXO{
				&account.UTXO{OutputID: bc.Hash{V0: 4}},
			},
		},
		{
			dbUtxos: map[string]*account.UTXO{
				string(account.StandardUTXOKey(bc.Hash{V0: 1})): &account.UTXO{
					OutputID: bc.Hash{V0: 1},
				},
				string(account.StandardUTXOKey(bc.Hash{V0: 1, V1: 2})): &account.UTXO{
					OutputID: bc.Hash{V0: 1, V1: 2},
				},
				string(account.StandardUTXOKey(bc.Hash{V0: 2})): &account.UTXO{
					OutputID: bc.Hash{V0: 2},
				},
				string(account.StandardUTXOKey(bc.Hash{V0: 2, V1: 2})): &account.UTXO{
					OutputID: bc.Hash{V0: 2, V1: 2},
				},
			},
			unconfirmedUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 6},
					ControlProgram: []byte{0x51},
				},
			},
			id:              "0000000000000002",
			unconfirmed:     false,
			isSmartContract: false,
			wantUtxos: []*account.UTXO{
				&account.UTXO{OutputID: bc.Hash{V0: 2}},
				&account.UTXO{OutputID: bc.Hash{V0: 2, V1: 2}},
			},
		},
		{
			dbUtxos: map[string]*account.UTXO{
				string(account.StandardUTXOKey(bc.Hash{V0: 3})): &account.UTXO{
					OutputID: bc.Hash{V0: 3},
				},
				string(account.ContractUTXOKey(bc.Hash{V0: 4})): &account.UTXO{
					OutputID: bc.Hash{V0: 4},
				},
			},
			unconfirmedUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 5},
					ControlProgram: []byte("smart contract"),
				},
				&account.UTXO{
					OutputID:       bc.Hash{V0: 6},
					ControlProgram: []byte{0x51},
				},
			},
			id:              "",
			unconfirmed:     true,
			isSmartContract: true,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 5},
					ControlProgram: []byte("smart contract"),
				},
				&account.UTXO{
					OutputID: bc.Hash{V0: 4},
				},
			},
		},
		{
			dbUtxos: map[string]*account.UTXO{
				string(account.StandardUTXOKey(bc.Hash{V0: 3})): &account.UTXO{
					OutputID: bc.Hash{V0: 3},
				},
				string(account.ContractUTXOKey(bc.Hash{V0: 4})): &account.UTXO{
					OutputID: bc.Hash{V0: 4},
				},
			},
			unconfirmedUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 5},
					ControlProgram: []byte("smart contract"),
				},
				&account.UTXO{
					OutputID:       bc.Hash{V0: 6},
					ControlProgram: []byte{0x51},
				},
			},
			id:              "",
			unconfirmed:     true,
			isSmartContract: false,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 6},
					ControlProgram: []byte{0x51},
				},
				&account.UTXO{
					OutputID: bc.Hash{V0: 3},
				},
			},
		},
	}

	w := &Wallet{DB: testDB}
	for i, c := range cases {
		for k, u := range c.dbUtxos {
			data, err := json.Marshal(u)
			if err != nil {
				t.Error(err)
			}
			testDB.Set([]byte(k), data)
		}

		w.AccountMgr = account.NewManager(testDB, nil)
		w.AccountMgr.AddUnconfirmedUtxo(c.unconfirmedUtxos)
		gotUtxos := w.GetAccountUtxos("", c.id, c.unconfirmed, c.isSmartContract)
		if !testutil.DeepEqual(gotUtxos, c.wantUtxos) {
			t.Errorf("case %d: got %v want %v", i, gotUtxos, c.wantUtxos)
		}

		for k := range c.dbUtxos {
			testDB.Delete([]byte(k))
		}
	}
}

//because can not pass by btm2.0 branch
func TestFilterAccountUtxo(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	cases := []struct {
		dbPrograms map[string]*account.CtrlProgram
		input      []*account.UTXO
		wantUtxos  []*account.UTXO
	}{
		{
			dbPrograms: map[string]*account.CtrlProgram{},
			input:      []*account.UTXO{},
			wantUtxos:  []*account.UTXO{},
		},
		{
			dbPrograms: map[string]*account.CtrlProgram{
				"436f6e74726163743a2a37a64a4e15a772ab43bf3f5956d0d1f353946496788e7f40d0ff1796286a6f": &account.CtrlProgram{
					AccountID: "testAccount",
					Address:   "testAddress",
					KeyIndex:  53,
					Change:    true,
				},
			},
			input: []*account.UTXO{
				&account.UTXO{
					ControlProgram: []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
				},
				&account.UTXO{
					ControlProgram: []byte{0x91},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         4,
				},
			},
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					ControlProgram:      []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:             bc.AssetID{V0: 1},
					Amount:              3,
					AccountID:           "testAccount",
					Address:             "testAddress",
					ControlProgramIndex: 53,
					Change:              true,
				},
			},
		},
		{
			dbPrograms: map[string]*account.CtrlProgram{},
			input: []*account.UTXO{
				&account.UTXO{
					ControlProgram: []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
				},
				&account.UTXO{
					ControlProgram: []byte{0x91},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
				},
			},
			wantUtxos: []*account.UTXO{},
		},
		{
			dbPrograms: map[string]*account.CtrlProgram{
				"436f6e74726163743a2a37a64a4e15a772ab43bf3f5956d0d1f353946496788e7f40d0ff1796286a6f": &account.CtrlProgram{
					AccountID: "testAccount",
					Address:   "testAddress",
					KeyIndex:  53,
					Change:    true,
				},
				"436f6e74726163743adb4d86262c12ba70d50b3ca3ae102d5682436243bd1e8c79569603f75675036a": &account.CtrlProgram{
					AccountID: "testAccount2",
					Address:   "testAddress2",
					KeyIndex:  72,
					Change:    false,
				},
			},
			input: []*account.UTXO{
				&account.UTXO{
					ControlProgram: []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
				},
				&account.UTXO{
					ControlProgram: []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         5,
				},
				&account.UTXO{
					ControlProgram: []byte{0x00, 0x14, 0xc6, 0xbf, 0x22, 0x19, 0x64, 0x2a, 0xc5, 0x9e, 0x5b, 0xe4, 0xeb, 0xdf, 0x5b, 0x22, 0x49, 0x56, 0xa7, 0x98, 0xa4, 0xdf},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         7,
				},
			},
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					ControlProgram:      []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:             bc.AssetID{V0: 1},
					Amount:              3,
					AccountID:           "testAccount",
					Address:             "testAddress",
					ControlProgramIndex: 53,
					Change:              true,
				},
				&account.UTXO{
					ControlProgram:      []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:             bc.AssetID{V0: 1},
					Amount:              5,
					AccountID:           "testAccount",
					Address:             "testAddress",
					ControlProgramIndex: 53,
					Change:              true,
				},
				&account.UTXO{
					ControlProgram:      []byte{0x00, 0x14, 0xc6, 0xbf, 0x22, 0x19, 0x64, 0x2a, 0xc5, 0x9e, 0x5b, 0xe4, 0xeb, 0xdf, 0x5b, 0x22, 0x49, 0x56, 0xa7, 0x98, 0xa4, 0xdf},
					AssetID:             bc.AssetID{V0: 1},
					Amount:              7,
					AccountID:           "testAccount2",
					Address:             "testAddress2",
					ControlProgramIndex: 72,
					Change:              false,
				},
			},
		},
	}

	w := &Wallet{DB: testDB}
	for i, c := range cases {
		for s, p := range c.dbPrograms {
			data, err := json.Marshal(p)
			if err != nil {
				t.Error(err)
			}
			key, err := hex.DecodeString(s)
			if err != nil {
				t.Error(err)
			}
			testDB.Set(key, data)
		}
		gotUtxos := w.filterAccountUtxo(c.input)
		sort.Slice(gotUtxos[:], func(i, j int) bool {
			return gotUtxos[i].Amount < gotUtxos[j].Amount
		})

		if !testutil.DeepEqual(gotUtxos, c.wantUtxos) {
			t.Errorf("case %d: got %v want %v", i, gotUtxos, c.wantUtxos)
		}
		for s := range c.dbPrograms {
			key, err := hex.DecodeString(s)
			if err != nil {
				t.Error(err)
			}
			testDB.Delete(key)
		}
	}
}

func TestTxInToUtxos(t *testing.T) {
	cases := []struct {
		tx         *types.Tx
		statusFail bool
		wantUtxos  []*account.UTXO
	}{
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewCoinbaseInput([]byte{0x51}),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 41250000000, []byte{0x51}, nil),
				},
			}),
			statusFail: false,
			wantUtxos:  []*account.UTXO{},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewIssuanceInput([]byte{}, 4125, []byte{0x51}, [][]byte{}, []byte{}),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 4125, []byte{0x51}, nil),
				},
			}),
			statusFail: false,
			wantUtxos:  []*account.UTXO{},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 1}, bc.AssetID{V0: 1}, 1, 1, []byte{0x51}, nil),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 2}, bc.AssetID{V0: 1}, 3, 2, []byte{0x52}, nil),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 3}, *consensus.BTMAssetID, 5, 3, []byte{0x53}, nil),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 4}, *consensus.BTMAssetID, 7, 4, []byte{0x54}, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(bc.AssetID{V0: 1}, 4, []byte{0x51}, nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 12, []byte{0x53}, nil),
				},
			}),
			statusFail: false,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x73, 0xf6, 0x02, 0xfc, 0x7c, 0x25, 0x98, 0xf4, 0xe6, 0xc5, 0xab, 0xa9, 0x31, 0x9f, 0x70, 0x3d, 0x64, 0x9a, 0xd6, 0x4a, 0xbd, 0x20, 0x5a, 0x72, 0x26, 0xba, 0x36, 0x5c, 0xc7, 0xb5, 0x92, 0xdf}),
					AssetID:        bc.AssetID{V0: 1},
					Amount:         1,
					ControlProgram: []byte{0x51},
					SourceID:       bc.Hash{V0: 1},
					SourcePos:      1,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0xb5, 0x53, 0xe5, 0xb3, 0xb2, 0xc0, 0x9d, 0xc1, 0x04, 0x02, 0xd8, 0xc3, 0x99, 0xb7, 0x0e, 0xe1, 0x14, 0xcc, 0xc8, 0xe9, 0x4d, 0x3d, 0x26, 0x51, 0x56, 0xd7, 0xf1, 0xf1, 0x03, 0xfa, 0xc1, 0x4b}),
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
					ControlProgram: []byte{0x52},
					SourceID:       bc.Hash{V0: 2},
					SourcePos:      2,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0xad, 0x0f, 0x82, 0xe6, 0x82, 0xee, 0xc0, 0xf8, 0xef, 0x21, 0xaa, 0x8b, 0xcd, 0xa0, 0x4b, 0x25, 0x58, 0x4d, 0x69, 0x2a, 0x6b, 0x4a, 0x17, 0x0e, 0xd3, 0x4c, 0x1e, 0x2c, 0x57, 0x74, 0x7a, 0x52}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         5,
					ControlProgram: []byte{0x53},
					SourceID:       bc.Hash{V0: 3},
					SourcePos:      3,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x27, 0x4e, 0x45, 0xa2, 0xd3, 0xe8, 0xe4, 0xb3, 0x54, 0x3c, 0xe2, 0x58, 0xc9, 0xcb, 0xd7, 0xed, 0xd9, 0xe2, 0x83, 0x78, 0xfb, 0x7a, 0xfa, 0xf3, 0x05, 0x29, 0xf9, 0xf1, 0x1e, 0xdb, 0x81, 0x7c}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         7,
					ControlProgram: []byte{0x54},
					SourceID:       bc.Hash{V0: 4},
					SourcePos:      4,
				},
			},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 3}, *consensus.BTMAssetID, 5, 3, []byte{0x53}, nil),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 4}, *consensus.BTMAssetID, 7, 4, []byte{0x54}, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(bc.AssetID{V0: 1}, 4, []byte{0x51}, nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 12, []byte{0x53}, nil),
				},
			}),
			statusFail: true,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0xad, 0x0f, 0x82, 0xe6, 0x82, 0xee, 0xc0, 0xf8, 0xef, 0x21, 0xaa, 0x8b, 0xcd, 0xa0, 0x4b, 0x25, 0x58, 0x4d, 0x69, 0x2a, 0x6b, 0x4a, 0x17, 0x0e, 0xd3, 0x4c, 0x1e, 0x2c, 0x57, 0x74, 0x7a, 0x52}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         5,
					ControlProgram: []byte{0x53},
					SourceID:       bc.Hash{V0: 3},
					SourcePos:      3,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x27, 0x4e, 0x45, 0xa2, 0xd3, 0xe8, 0xe4, 0xb3, 0x54, 0x3c, 0xe2, 0x58, 0xc9, 0xcb, 0xd7, 0xed, 0xd9, 0xe2, 0x83, 0x78, 0xfb, 0x7a, 0xfa, 0xf3, 0x05, 0x29, 0xf9, 0xf1, 0x1e, 0xdb, 0x81, 0x7c}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         7,
					ControlProgram: []byte{0x54},
					SourceID:       bc.Hash{V0: 4},
					SourcePos:      4,
				},
			},
		},
	}

	for i, c := range cases {
		if gotUtxos := txInToUtxos(c.tx); !testutil.DeepEqual(gotUtxos, c.wantUtxos) {
			for k, v := range gotUtxos {
				data, _ := json.Marshal(v)
				fmt.Println(k, string(data))
			}
			for k, v := range c.wantUtxos {
				data, _ := json.Marshal(v)
				fmt.Println(k, string(data))
			}
			t.Errorf("case %d: got %v want %v", i, gotUtxos, c.wantUtxos)
		}
	}
}

func TestTxOutToUtxos(t *testing.T) {
	cases := []struct {
		tx          *types.Tx
		statusFail  bool
		vaildHeight uint64
		wantUtxos   []*account.UTXO
	}{
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewCoinbaseInput([]byte{0x51}),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 41250000000, []byte{0x51}, nil),
				},
			}),
			statusFail:  false,
			vaildHeight: 98,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x9c, 0xab, 0x55, 0xdc, 0x72, 0xb1, 0x42, 0x6d, 0x2a, 0x41, 0x92, 0xc3, 0x40, 0x32, 0x29, 0xf4, 0xa4, 0x11, 0xae, 0x54, 0x41, 0x54, 0x1a, 0xfe, 0x7c, 0x93, 0x4b, 0x8f, 0x6c, 0x61, 0x69, 0x9f}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         41250000000,
					ControlProgram: []byte{0x51},
					SourceID:       bc.NewHash([32]byte{0xb4, 0x7e, 0x94, 0x31, 0x88, 0xfe, 0xd3, 0xe9, 0xac, 0x99, 0x7c, 0xfc, 0x99, 0x6d, 0xd7, 0x4d, 0x04, 0x10, 0x77, 0xcb, 0x1c, 0xf8, 0x95, 0x14, 0x00, 0xe3, 0x42, 0x00, 0x8d, 0x05, 0xec, 0xdc}),
					SourcePos:      0,
					ValidHeight:    98,
				},
			},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 1}, bc.AssetID{V0: 1}, 5, 1, []byte{0x51}, nil),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 2}, *consensus.BTMAssetID, 7, 1, []byte{0x51}, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(bc.AssetID{V0: 1}, 2, []byte{0x51}, nil),
					types.NewOriginalTxOutput(bc.AssetID{V0: 1}, 3, []byte{0x52}, nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 2, []byte{0x53}, nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 5, []byte{0x54}, nil),
				},
			}),
			statusFail:  false,
			vaildHeight: 0,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x49, 0x33, 0x66, 0x49, 0x4b, 0xaa, 0x57, 0x26, 0xc7, 0x21, 0x74, 0x75, 0x4e, 0x15, 0x59, 0xa4, 0x24, 0xa1, 0x92, 0xda, 0xb1, 0x88, 0x8f, 0xea, 0x51, 0xaf, 0xcf, 0x95, 0x21, 0xab, 0xe4, 0xe2}),
					AssetID:        bc.AssetID{V0: 1},
					Amount:         2,
					ControlProgram: []byte{0x51},
					SourceID:       bc.NewHash([32]byte{0x45, 0xe7, 0xf8, 0x36, 0x9c, 0x5c, 0x79, 0x9c, 0xf5, 0xd7, 0x5b, 0x95, 0x02, 0x77, 0x5e, 0xd6, 0x39, 0x08, 0x03, 0x97, 0x6f, 0xf7, 0x47, 0x54, 0xc6, 0xa4, 0xcf, 0x6e, 0x5b, 0x0b, 0x4e, 0x0b}),
					SourcePos:      0,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0xca, 0x72, 0xfb, 0x89, 0xfc, 0x83, 0xda, 0x24, 0x01, 0x63, 0x8f, 0xac, 0x5e, 0xd2, 0x84, 0xad, 0x27, 0x6d, 0x41, 0xc0, 0x01, 0x45, 0x93, 0x08, 0x3e, 0x37, 0x8f, 0xe8, 0x00, 0x46, 0x6a, 0xfd}),
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
					ControlProgram: []byte{0x52},
					SourceID:       bc.NewHash([32]byte{0x45, 0xe7, 0xf8, 0x36, 0x9c, 0x5c, 0x79, 0x9c, 0xf5, 0xd7, 0x5b, 0x95, 0x02, 0x77, 0x5e, 0xd6, 0x39, 0x08, 0x03, 0x97, 0x6f, 0xf7, 0x47, 0x54, 0xc6, 0xa4, 0xcf, 0x6e, 0x5b, 0x0b, 0x4e, 0x0b}),
					SourcePos:      1,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0xb1, 0x7a, 0xbf, 0xa, 0x36, 0x78, 0x76, 0xa3, 0x59, 0xfd, 0xa8, 0x1c, 0x50, 0xba, 0xc4, 0x5f, 0x60, 0xbb, 0x79, 0x6, 0x79, 0x6c, 0x26, 0x1b, 0x7e, 0x73, 0x87, 0x54, 0xbd, 0xbf, 0xe5, 0xd8}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         2,
					ControlProgram: []byte{0x53},
					SourceID:       bc.NewHash([32]byte{0x45, 0xe7, 0xf8, 0x36, 0x9c, 0x5c, 0x79, 0x9c, 0xf5, 0xd7, 0x5b, 0x95, 0x02, 0x77, 0x5e, 0xd6, 0x39, 0x08, 0x03, 0x97, 0x6f, 0xf7, 0x47, 0x54, 0xc6, 0xa4, 0xcf, 0x6e, 0x5b, 0x0b, 0x4e, 0x0b}),
					SourcePos:      2,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x24, 0xb1, 0x96, 0x33, 0x99, 0xaa, 0xee, 0xbd, 0x88, 0xcb, 0xa2, 0x7f, 0x66, 0xfa, 0x87, 0xa8, 0xce, 0x3, 0x83, 0x7e, 0x18, 0x8e, 0x36, 0xae, 0x92, 0xda, 0xe0, 0x99, 0xa5, 0xec, 0x8f, 0x08}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         5,
					ControlProgram: []byte{0x54},
					SourceID:       bc.NewHash([32]byte{0x45, 0xe7, 0xf8, 0x36, 0x9c, 0x5c, 0x79, 0x9c, 0xf5, 0xd7, 0x5b, 0x95, 0x02, 0x77, 0x5e, 0xd6, 0x39, 0x08, 0x03, 0x97, 0x6f, 0xf7, 0x47, 0x54, 0xc6, 0xa4, 0xcf, 0x6e, 0x5b, 0x0b, 0x4e, 0x0b}),
					SourcePos:      3,
				},
			},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 1}, bc.AssetID{V0: 1}, 5, 1, []byte{0x51}, nil),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 2}, *consensus.BTMAssetID, 7, 1, []byte{0x51}, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(bc.AssetID{V0: 1}, 2, []byte{0x51}, nil),
					types.NewOriginalTxOutput(bc.AssetID{V0: 1}, 3, []byte{0x52}, nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 2, []byte{0x53}, nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 5, []byte{0x54}, nil),
				},
			}),
			statusFail:  true,
			vaildHeight: 0,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x49, 0x33, 0x66, 0x49, 0x4b, 0xaa, 0x57, 0x26, 0xc7, 0x21, 0x74, 0x75, 0x4e, 0x15, 0x59, 0xa4, 0x24, 0xa1, 0x92, 0xda, 0xb1, 0x88, 0x8f, 0xea, 0x51, 0xaf, 0xcf, 0x95, 0x21, 0xab, 0xe4, 0xe2}),
					AssetID:        bc.AssetID{V0: 1},
					Amount:         2,
					ControlProgram: []byte{0x51},
					SourceID:       bc.NewHash([32]byte{0x45, 0xe7, 0xf8, 0x36, 0x9c, 0x5c, 0x79, 0x9c, 0xf5, 0xd7, 0x5b, 0x95, 0x02, 0x77, 0x5e, 0xd6, 0x39, 0x08, 0x03, 0x97, 0x6f, 0xf7, 0x47, 0x54, 0xc6, 0xa4, 0xcf, 0x6e, 0x5b, 0x0b, 0x4e, 0x0b}),
					SourcePos:      0,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0xca, 0x72, 0xfb, 0x89, 0xfc, 0x83, 0xda, 0x24, 0x01, 0x63, 0x8f, 0xac, 0x5e, 0xd2, 0x84, 0xad, 0x27, 0x6d, 0x41, 0xc0, 0x01, 0x45, 0x93, 0x08, 0x3e, 0x37, 0x8f, 0xe8, 0x00, 0x46, 0x6a, 0xfd}),
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
					ControlProgram: []byte{0x52},
					SourceID:       bc.NewHash([32]byte{0x45, 0xe7, 0xf8, 0x36, 0x9c, 0x5c, 0x79, 0x9c, 0xf5, 0xd7, 0x5b, 0x95, 0x02, 0x77, 0x5e, 0xd6, 0x39, 0x08, 0x03, 0x97, 0x6f, 0xf7, 0x47, 0x54, 0xc6, 0xa4, 0xcf, 0x6e, 0x5b, 0x0b, 0x4e, 0x0b}),
					SourcePos:      1,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0xb1, 0x7a, 0xbf, 0x0a, 0x36, 0x78, 0x76, 0xa3, 0x59, 0xfd, 0xa8, 0x1c, 0x50, 0xba, 0xc4, 0x5f, 0x60, 0xbb, 0x79, 0x06, 0x79, 0x6c, 0x26, 0x1b, 0x7e, 0x73, 0x87, 0x54, 0xbd, 0xbf, 0xe5, 0xd8}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         2,
					ControlProgram: []byte{0x53},
					SourceID:       bc.NewHash([32]byte{0x45, 0xe7, 0xf8, 0x36, 0x9c, 0x5c, 0x79, 0x9c, 0xf5, 0xd7, 0x5b, 0x95, 0x02, 0x77, 0x5e, 0xd6, 0x39, 0x08, 0x03, 0x97, 0x6f, 0xf7, 0x47, 0x54, 0xc6, 0xa4, 0xcf, 0x6e, 0x5b, 0x0b, 0x4e, 0x0b}),
					SourcePos:      2,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x24, 0xb1, 0x96, 0x33, 0x99, 0xaa, 0xee, 0xbd, 0x88, 0xcb, 0xa2, 0x7f, 0x66, 0xfa, 0x87, 0xa8, 0xce, 0x03, 0x83, 0x7e, 0x18, 0x8e, 0x36, 0xae, 0x92, 0xda, 0xe0, 0x99, 0xa5, 0xec, 0x8f, 0x08}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         5,
					ControlProgram: []byte{0x54},
					SourceID:       bc.NewHash([32]byte{0x45, 0xe7, 0xf8, 0x36, 0x9c, 0x5c, 0x79, 0x9c, 0xf5, 0xd7, 0x5b, 0x95, 0x02, 0x77, 0x5e, 0xd6, 0x39, 0x08, 0x03, 0x97, 0x6f, 0xf7, 0x47, 0x54, 0xc6, 0xa4, 0xcf, 0x6e, 0x5b, 0x0b, 0x4e, 0x0b}),
					SourcePos:      3,
				},
			},
		},
	}

	for i, c := range cases {
		if gotUtxos := txOutToUtxos(c.tx, c.vaildHeight); !testutil.DeepEqual(gotUtxos, c.wantUtxos) {
			for k, v := range gotUtxos {

				data, _ := json.Marshal(v)
				fmt.Println("got:", k, string(data))
			}
			for k, v := range c.wantUtxos {
				data, _ := json.Marshal(v)
				fmt.Println("want:", k, string(data))
			}
			t.Errorf("case %d: got %v want %v", i, gotUtxos, c.wantUtxos)
		}
	}
}
