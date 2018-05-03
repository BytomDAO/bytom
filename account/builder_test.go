package account

import (
	"testing"

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
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  300,
					},
					AccountID: "test_account2",
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
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  300,
					},
					AccountID: "test_account2",
				}),
			},
			testActionCount: 3,
			wantActionCount: 3,
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
