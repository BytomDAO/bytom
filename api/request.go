package api

import (
	"context"
	"strings"

	"github.com/bytom/consensus"
	"github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc/types"
)

var (
	errBadActionType = errors.New("bad action type")
	errBadAction     = errors.New("bad action object")
)

// BuildRequest is main struct when building transactions
type BuildRequest struct {
	Tx        *types.TxData            `json:"base_transaction"`
	Actions   []map[string]interface{} `json:"actions"`
	TTL       json.Duration            `json:"ttl"`
	TimeRange uint64                   `json:"time_range"`
}

func (a *API) filterAliases(ctx context.Context, br *BuildRequest) error {
	for i, m := range br.Actions {
		id, _ := m["asset_id"].(string)
		alias, _ := m["asset_alias"].(string)
		alias = strings.ToUpper(alias)
		if id == "" && alias != "" {
			switch alias {
			case consensus.BTMAlias:
				m["asset_id"] = consensus.BTMAssetID.String()
			default:
				asset, err := a.wallet.AssetReg.FindByAlias(alias)
				if err != nil {
					return errors.WithDetailf(err, "invalid asset alias %s on action %d", alias, i)
				}
				m["asset_id"] = asset.AssetID.String()
			}
		}

		id, _ = m["account_id"].(string)
		alias, _ = m["account_alias"].(string)
		if id == "" && alias != "" {
			acc, err := a.wallet.AccountMgr.FindByAlias(ctx, alias)
			if err != nil {
				return errors.WithDetailf(err, "invalid account alias %s on action %d", alias, i)
			}
			m["account_id"] = acc.ID
		}
	}
	return nil
}
