package blockchain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
)

func TestMergeActions(t *testing.T) {
	cases := []struct {
		buildStr    string
		actionCount int
		wantBTM     int64
		wantOther   int64
	}{
		{
			`{"actions": [
			{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":100, "account_id": "123"},
				{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount": 200,"account_id": "123"},
				{"type": "control_receiver", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": 200, "receiver":{"control_program": "program","expires_at":"2017"}}
			]}`,
			2,
			300,
			0,
		},
		{
			`{"actions": [
			{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":100, "account_id": "123"},
				{"type": "spend_account", "asset_id": "43c6946d092b2959c1a82e90b282c68fca63e66de289048f6acd6cea9383c79c","amount": 200,"account_id": "123"},
				{"type": "control_receiver", "asset_id": "43c6946d092b2959c1a82e90b282c68fca63e66de289048f6acd6cea9383c79c", "amount": 200, "receiver":{"control_program": "program","expires_at":"2017"}}
			]}`,
			3,
			100,
			200,
		}, {
			`{"actions": [
			{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":100, "account_id": "123"},
				{"type": "spend_account", "asset_id": "43c6946d092b2959c1a82e90b282c68fca63e66de289048f6acd6cea9383c79c","amount": 200,"account_id": "123"},
				{"type": "spend_account", "asset_id": "43c6946d092b2959c1a82e90b282c68fca63e66de289048f6acd6cea9383c79c","amount": 300,"account_id": "123"},
				{"type": "control_receiver", "asset_id": "43c6946d092b2959c1a82e90b282c68fca63e66de289048f6acd6cea9383c79c", "amount": 500, "receiver":{"control_program": "program","expires_at":"2017"}}
			]}`,
			3,
			100,
			500,
		},
		{
			`{"actions": [
			{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":100, "account_id": "123"},
				{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount": 200,"account_id": "123"},
				{"type": "spend_account", "asset_id": "43c6946d092b2959c1a82e90b282c68fca63e66de289048f6acd6cea9383c79c","amount": 200,"account_id": "123"},
				{"type": "spend_account", "asset_id": "43c6946d092b2959c1a82e90b282c68fca63e66de289048f6acd6cea9383c79c","amount": 300,"account_id": "123"},
				{"type": "control_receiver", "asset_id": "43c6946d092b2959c1a82e90b282c68fca63e66de289048f6acd6cea9383c79c", "amount": 500, "receiver":{"control_program": "program","expires_at":"2017"}}
			]}`,
			3,
			300,
			500,
		},
	}

	for i, c := range cases {
		BuildReq := &BuildRequest{}

		if err := json.Unmarshal([]byte(c.buildStr), BuildReq); err != nil {
			t.Fatal(err)
		}

		for _, m := range BuildReq.Actions {
			amount := m["amount"].(float64)
			m["amount"] = json.Number(fmt.Sprintf("%v", amount))
		}

		actions := mergeActions(BuildReq)

		if len(actions) != c.actionCount {
			t.Fatalf("got error count %d, want %d", len(actions), c.actionCount)
		}

		for _, a := range actions {
			actionType := a["type"].(string)
			assetID := a["asset_id"].(string)
			if actionType == "spend_account" {
				amountStr := fmt.Sprintf("%v", a["amount"])
				amount, _ := strconv.ParseInt(amountStr, 10, 64)

				switch assetID {
				case "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff":
					if amount != c.wantBTM {
						t.Fatalf("index %d, get error amount %v, want %d", i, amount, c.wantBTM)
					}
				case "43c6946d092b2959c1a82e90b282c68fca63e66de289048f6acd6cea9383c79c":
					if amount != c.wantOther {
						t.Fatalf("index %d, get error amount %v, want %d", i, amount, c.wantOther)
					}
				default:
					t.Fatalf("no %s in test cases", assetID)
				}
			}
		}
	}
}
