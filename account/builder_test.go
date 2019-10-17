package account

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
)

func TestReserveBtmUtxoChain(t *testing.T) {
	txbuilder.ChainTxUtxoNum = 3
	utxos := []*UTXO{}
	m := mockAccountManager(t)
	for i := uint64(1); i <= 20; i++ {
		utxo := &UTXO{
			OutputID:  bc.Hash{V0: i},
			AccountID: "TestAccountID",
			AssetID:   *consensus.BTMAssetID,
			Amount:    i * txbuilder.ChainTxMergeGas,
		}
		utxos = append(utxos, utxo)

		data, err := json.Marshal(utxo)
		if err != nil {
			t.Fatal(err)
		}

		m.db.Set(StandardUTXOKey(utxo.OutputID), data)
	}

	cases := []struct {
		amount uint64
		want   []uint64
		err    bool
	}{
		{
			amount: 1 * txbuilder.ChainTxMergeGas,
			want:   []uint64{1},
		},
		{
			amount: 888888 * txbuilder.ChainTxMergeGas,
			want:   []uint64{},
			err:    true,
		},
		{
			amount: 7 * txbuilder.ChainTxMergeGas,
			want:   []uint64{4, 3, 1},
		},
		{
			amount: 15 * txbuilder.ChainTxMergeGas,
			want:   []uint64{5, 4, 3, 2, 1, 6},
		},
		{
			amount: 163 * txbuilder.ChainTxMergeGas,
			want:   []uint64{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 2, 1, 3},
		},
	}

	for i, c := range cases {
		m.utxoKeeper.expireReservation(time.Unix(999999999, 0))
		utxos, err := m.reserveBtmUtxoChain(&txbuilder.TemplateBuilder{}, "TestAccountID", c.amount, false)

		if err != nil != c.err {
			t.Fatalf("case %d got err %v want err = %v", i, err, c.err)
		}

		got := []uint64{}
		for _, utxo := range utxos {
			got = append(got, utxo.Amount/txbuilder.ChainTxMergeGas)
		}

		if !testutil.DeepEqual(got, c.want) {
			t.Fatalf("case %d got %d want %d", i, got, c.want)
		}
	}

}

func TestBuildBtmTxChain(t *testing.T) {
	txbuilder.ChainTxUtxoNum = 3
	m := mockAccountManager(t)
	cases := []struct {
		inputUtxo  []uint64
		wantInput  [][]uint64
		wantOutput [][]uint64
		wantUtxo   uint64
	}{
		{
			inputUtxo:  []uint64{5},
			wantInput:  [][]uint64{},
			wantOutput: [][]uint64{},
			wantUtxo:   5 * txbuilder.ChainTxMergeGas,
		},
		{
			inputUtxo: []uint64{5, 4},
			wantInput: [][]uint64{
				[]uint64{5, 4},
			},
			wantOutput: [][]uint64{
				[]uint64{8},
			},
			wantUtxo: 8 * txbuilder.ChainTxMergeGas,
		},
		{
			inputUtxo: []uint64{5, 4, 1, 1},
			wantInput: [][]uint64{
				[]uint64{5, 4, 1},
				[]uint64{1, 9},
			},
			wantOutput: [][]uint64{
				[]uint64{9},
				[]uint64{9},
			},
			wantUtxo: 9 * txbuilder.ChainTxMergeGas,
		},
		{
			inputUtxo: []uint64{22, 123, 53, 234, 23, 4, 2423, 24, 23, 43, 34, 234, 234, 24},
			wantInput: [][]uint64{
				[]uint64{22, 123, 53},
				[]uint64{234, 23, 4},
				[]uint64{2423, 24, 23},
				[]uint64{43, 34, 234},
				[]uint64{234, 24, 197},
				[]uint64{260, 2469, 310},
				[]uint64{454, 3038},
			},
			wantOutput: [][]uint64{
				[]uint64{197},
				[]uint64{260},
				[]uint64{2469},
				[]uint64{310},
				[]uint64{454},
				[]uint64{3038},
				[]uint64{3491},
			},
			wantUtxo: 3491 * txbuilder.ChainTxMergeGas,
		},
	}

	acct, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, "testAccount", signers.BIP0044)
	if err != nil {
		t.Fatal(err)
	}

	acp, err := m.CreateAddress(acct.ID, false)
	if err != nil {
		t.Fatal(err)
	}

	for caseIndex, c := range cases {
		utxos := []*UTXO{}
		for _, amount := range c.inputUtxo {
			utxos = append(utxos, &UTXO{
				Amount:         amount * txbuilder.ChainTxMergeGas,
				AssetID:        *consensus.BTMAssetID,
				Address:        acp.Address,
				ControlProgram: acp.ControlProgram,
			})
		}

		tpls, gotUtxo, err := m.buildBtmTxChain(utxos, acct.Signer)
		if err != nil {
			t.Fatal(err)
		}

		for i, tpl := range tpls {
			gotInput := []uint64{}
			for _, input := range tpl.Transaction.Inputs {
				gotInput = append(gotInput, input.Amount()/txbuilder.ChainTxMergeGas)
			}

			gotOutput := []uint64{}
			for _, output := range tpl.Transaction.Outputs {
				gotOutput = append(gotOutput, output.Amount/txbuilder.ChainTxMergeGas)
			}

			if !testutil.DeepEqual(c.wantInput[i], gotInput) {
				t.Fatalf("case %d tx %d input got %d want %d", caseIndex, i, gotInput, c.wantInput[i])
			}
			if !testutil.DeepEqual(c.wantOutput[i], gotOutput) {
				t.Fatalf("case %d tx %d output got %d want %d", caseIndex, i, gotOutput, c.wantOutput[i])
			}
		}

		if c.wantUtxo != gotUtxo.Amount {
			t.Fatalf("case %d got utxo=%d want utxo=%d", caseIndex, gotUtxo.Amount, c.wantUtxo)
		}
	}

}

func TestMergeSpendAction(t *testing.T) {
	testBTM := &bc.AssetID{}
	if err := testBTM.UnmarshalText([]byte("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")); err != nil {
		t.Fatal(err)
	}

	testAssetID1 := &bc.AssetID{}
	if err := testAssetID1.UnmarshalText([]byte("50ec80b6bc48073f6aa8fa045131a71213c33f3681203b15ddc2e4b81f1f4730")); err != nil {
		t.Fatal(err)
	}

	testAssetID2 := &bc.AssetID{}
	if err := testAssetID2.UnmarshalText([]byte("43c6946d092b2959c1a82e90b282c68fca63e66de289048f6acd6cea9383c79c")); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		testActions     []txbuilder.Action
		wantActions     []txbuilder.Action
		testActionCount int
		wantActionCount int
	}{
		{
			testActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  200,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  600,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
			},
			testActionCount: 4,
			wantActionCount: 2,
		},
		{
			testActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  600,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
			},
			testActionCount: 4,
			wantActionCount: 2,
		},
		{
			testActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  800,
					},
					AccountID: "test_account",
				}),
			},
			testActionCount: 5,
			wantActionCount: 3,
		},
		{
			testActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account1",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  600,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account1",
				}),
			},
			testActionCount: 4,
			wantActionCount: 3,
		},
		{
			testActions: []txbuilder.Action{
				txbuilder.Action(&spendUTXOAction{
					OutputID: &bc.Hash{V0: 128},
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&spendUTXOAction{
					OutputID: &bc.Hash{V0: 256},
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  300,
					},
					AccountID: "test_account2",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&spendUTXOAction{
					OutputID: &bc.Hash{V0: 128},
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&spendUTXOAction{
					OutputID: &bc.Hash{V0: 256},
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  300,
					},
					AccountID: "test_account2",
				}),
			},
			testActionCount: 5,
			wantActionCount: 5,
		},
	}

	for _, c := range cases {
		gotActions := MergeSpendAction(c.testActions)

		gotMap := make(map[string]uint64)
		wantMap := make(map[string]uint64)
		for _, got := range gotActions {
			switch got := got.(type) {
			case *spendAction:
				gotKey := got.AssetId.String() + got.AccountID
				gotMap[gotKey] = got.Amount
			default:
				continue
			}
		}

		for _, want := range c.wantActions {
			switch want := want.(type) {
			case *spendAction:
				wantKey := want.AssetId.String() + want.AccountID
				wantMap[wantKey] = want.Amount
			default:
				continue
			}
		}

		for key := range gotMap {
			if gotMap[key] != wantMap[key] {
				t.Fatalf("gotMap[%s]=%v, wantMap[%s]=%v", key, gotMap[key], key, wantMap[key])
			}
		}

		if len(gotActions) != c.wantActionCount {
			t.Fatalf("number of gotActions=%d, wantActions=%d", len(gotActions), c.wantActionCount)
		}
	}
}

func TestCalcMergeGas(t *testing.T) {
	txbuilder.ChainTxUtxoNum = 10
	cases := []struct {
		utxoNum int
		gas     uint64
	}{
		{
			utxoNum: 0,
			gas:     0,
		},
		{
			utxoNum: 1,
			gas:     0,
		},
		{
			utxoNum: 9,
			gas:     txbuilder.ChainTxMergeGas,
		},
		{
			utxoNum: 10,
			gas:     txbuilder.ChainTxMergeGas,
		},
		{
			utxoNum: 11,
			gas:     txbuilder.ChainTxMergeGas * 2,
		},
		{
			utxoNum: 20,
			gas:     txbuilder.ChainTxMergeGas * 3,
		},
		{
			utxoNum: 21,
			gas:     txbuilder.ChainTxMergeGas * 3,
		},
		{
			utxoNum: 74,
			gas:     txbuilder.ChainTxMergeGas * 9,
		},
	}

	for i, c := range cases {
		gas := calcMergeGas(c.utxoNum)
		if gas != c.gas {
			t.Fatalf("case %d got %d want %d", i, gas, c.gas)
		}
	}
}
