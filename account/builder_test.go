package account

import (
	"encoding/json"
	"testing"

	"encoding/hex"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/protocol/bc"
)

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

func TestSpendUTXOArguments(t *testing.T) {
	cases := []struct {
		rawAction  string
		wantResult bool
	}{
		{
			rawAction: `{ "type": "spend_account_unspent_output", "output_id": "e304de887423e4e684e483f5ae65236d47018b56cac94ef3fb8b5dd40c897e11",
				"arguments": [{"type": "raw_tx_signature", "raw_data": {"derivation_path": ["010100000000000000", "0100000000000000"],
	            "xpub": "ba76bb52574b3f40315f2c01f1818a9072ced56e9d4b68acbef56a4d0077d08e5e34837963e4cdc54eb251aa34aad01e6ae48b140f6a2743fbb0a0abd9cf8aac"}}]}`,
			wantResult: true,
		},
		{
			rawAction: `{ "type": "spend_account_unspent_output", "output_id": "8669b5c2e0701ec1ca45cd413e46c4f1d5f794f9d9144f904f3e7da8c68c6410",
				"arguments": [{"type": "data", "raw_data": {"value": "7468697320697320612074657374"}}]}`,
			wantResult: true,
		},
		{
			rawAction: `{ "type": "spend_account_unspent_output", "output_id": "8669b5c2e0701ec1ca45cd413e46c4f1d5f794f9d9144f904f3e7da8c68c6410",
				"arguments": [{"type": "signature", "raw_data": {"value": "7468697320697320612074657374"}}]}`,
			wantResult: false,
		},
	}

	for _, c := range cases {
		var spendUTXOReq spendUTXOAction
		if err := json.Unmarshal([]byte(c.rawAction), &spendUTXOReq); err != nil {
			t.Fatalf("unmarshal spendUTXOAction error:%v", err)
		}

		if spendUTXOReq.Arguments != nil {
			for _, arg := range spendUTXOReq.Arguments {
				switch arg.Type {
				case "raw_tx_signature":
					rawTxSig := &RawTxSigArgument{}
					if err := json.Unmarshal(arg.RawData, rawTxSig); err != nil {
						t.Fatalf("unmarshal RawTxSigArgument error:%v", err)
					}

					gotResult := false
					if hex.EncodeToString(rawTxSig.RootXPub[:]) == "ba76bb52574b3f40315f2c01f1818a9072ced56e9d4b68acbef56a4d0077d08e5e34837963e4cdc54eb251aa34aad01e6ae48b140f6a2743fbb0a0abd9cf8aac" &&
						hex.EncodeToString(rawTxSig.Path[0]) == "010100000000000000" && hex.EncodeToString(rawTxSig.Path[1]) == "0100000000000000" {
						gotResult = true
					}

					if gotResult != c.wantResult {
						t.Fatalf("Unmarshal RawTxSigArgument result is not right: %v", rawTxSig)
					}

				case "data":
					data := &DataArgument{}
					if err := json.Unmarshal(arg.RawData, data); err != nil {
						t.Fatalf("unmarshal DataArgument error:%v", err)
					}

					gotResult := false
					if data.Value == "7468697320697320612074657374" {
						gotResult = true
					}

					if gotResult != c.wantResult {
						t.Fatalf("Unmarshal DataArgument result is not right: %v", data)
					}

				default:
					if c.wantResult {
						t.Fatalf("contract argument type [%v] is not exist", arg.Type)
					}
				}
			}
		}
	}
}
