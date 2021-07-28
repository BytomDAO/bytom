package apinode

import (
	"encoding/json"
	"testing"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
)

func buildTxRequest(accountID string, outputs map[string]uint64) ([]byte, error) {
	totalBTM := uint64(10000000)
	actions := []interface{}{}
	for address, amount := range outputs {
		actions = append(actions, &ControlAddressAction{
			Address:     address,
			AssetAmount: &bc.AssetAmount{AssetId: consensus.BTMAssetID, Amount: amount},
		})
		totalBTM += amount
	}

	actions = append(actions, &SpendAccountAction{
		AccountID:   accountID,
		AssetAmount: &bc.AssetAmount{AssetId: consensus.BTMAssetID, Amount: totalBTM},
	})
	payload, err := json.Marshal(&buildTxReq{Actions: actions})
	if err != nil {
		return nil, errors.Wrap(err, "Marshal spend request")
	}

	return payload, nil
}

type args struct {
	accountID string
	outputs   map[string]uint64
}

func TestBuildTxRequest(t *testing.T) {
	cases := []struct {
		args       args
		optionWant map[string]bool
	}{
		{
			args: args{
				accountID: "9bb77612-350e-4d53-81e2-525b28247ba5",
				outputs:   map[string]uint64{"sp1qlryy65a5apylphqp6axvhx7nd6y2zlexuvn7gf": 100},
			},
			optionWant: map[string]bool{`{"actions":[{"type":"control_address","address":"sp1qlryy65a5apylphqp6axvhx7nd6y2zlexuvn7gf","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":100},{"type":"spend_account","account_id":"9bb77612-350e-4d53-81e2-525b28247ba5","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":10000100}]}`: true},
		},
		{
			args: args{
				accountID: "9bb77612-350e-4d53-81e2-525b28247ba5",
				outputs:   map[string]uint64{"sp1qlryy65a5apylphqp6axvhx7nd6y2zlexuvn7gf": 100, "sp1qcgtxkhfzytul4lfttwex3skfqhm0tg6ms9da28": 200},
			},
			optionWant: map[string]bool{`{"actions":[{"type":"control_address","address":"sp1qlryy65a5apylphqp6axvhx7nd6y2zlexuvn7gf","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":100},{"type":"control_address","address":"sp1qcgtxkhfzytul4lfttwex3skfqhm0tg6ms9da28","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":200},{"type":"spend_account","account_id":"9bb77612-350e-4d53-81e2-525b28247ba5","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":10000300}]}`: true, `{"actions":[{"type":"control_address","address":"sp1qcgtxkhfzytul4lfttwex3skfqhm0tg6ms9da28","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":200},{"type":"control_address","address":"sp1qlryy65a5apylphqp6axvhx7nd6y2zlexuvn7gf","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":100},{"type":"spend_account","account_id":"9bb77612-350e-4d53-81e2-525b28247ba5","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":10000300}]}`: true},
		},
	}

	for i, c := range cases {
		tx, err := buildTxRequest(c.args.accountID, c.args.outputs)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := c.optionWant[string(tx)]; !ok {
			t.Fatal(i, string(tx))
		}
	}
}
