package blockchain

import (
	"context"

	"github.com/bytom/consensus"
	"github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc/legacy"
)

var (
	errBadActionType = errors.New("bad action type")
	errBadAction     = errors.New("bad action object")
)

//BuildRequest is main struct when building transactions
type BuildRequest struct {
	Tx      *legacy.TxData           `json:"base_transaction"`
	Actions []map[string]interface{} `json:"actions"`
	TTL     json.Duration            `json:"ttl"`
}

func (bcr *BlockchainReactor) filterAliases(ctx context.Context, br *BuildRequest) error {
	for i, m := range br.Actions {
		id, _ := m["asset_id"].(string)
		alias, _ := m["asset_alias"].(string)
		if id == "" && alias != "" {
			switch alias {
			case consensus.BTMAlias:
				m["asset_id"] = consensus.BTMAssetID.String()
			default:
				id, err := bcr.assets.GetIDByAlias(alias)
				if err != nil {
					return errors.WithDetailf(err, "invalid asset alias %s on action %d", alias, i)
				}
				m["asset_id"] = id
			}
		}

		id, _ = m["account_id"].(string)
		alias, _ = m["account_alias"].(string)
		if id == "" && alias != "" {
			acc, err := bcr.accounts.FindByAlias(ctx, alias)
			if err != nil {
				return errors.WithDetailf(err, "invalid account alias %s on action %d", alias, i)
			}
			m["account_id"] = acc.ID
		}

		contractOperator, _ := m["contract_operator"].(string)
		if contractOperator == "" {
			continue
		}

		if bcr.accounts.GetAliasByID(contractOperator) != "" {
			//contractOperator is ID
			m["contract_operator"] = contractOperator
		} else {
			//contractOperator is Alias
			acc, err := bcr.accounts.FindByAlias(ctx, contractOperator)
			if err != nil {
				return errors.WithDetailf(err, "invalid contract operator alias %s on action %d", contractOperator, i)
			}
			m["contract_operator"] = acc.ID
		}
	}
	return nil
}

// getContractAccountID can acquire the accountID and contractProgram by parsing BuildRequest's Actions,
// when BuildRequest's Actions contain more than one accountInfo("account_id" or "account_alias"), the accountInfo is same
func (bcr *BlockchainReactor) getContractAccountID(ctx context.Context, br *BuildRequest) (accountID string, contractProgram string, err error) {
	for _, m := range br.Actions {
		id, _ := m["account_id"].(string)
		alias, _ := m["account_alias"].(string)
		contractProgram, _ = m["control_program"].(string)

		if id != "" {
			accountID = id
		} else if id == "" && alias != "" {
			acc, errMsg := bcr.accounts.FindByAlias(ctx, alias)
			if errMsg != nil {
				err = errors.WithDetailf(errMsg, "invalid account alias %s", alias)
				return
			}
			accountID = acc.ID
		}
	}

	if accountID == "" {
		err = errors.New("account is empty")
	}

	if contractProgram == "" {
		err = errors.New("contract Program is empty")
	}

	return
}
