package api

import (
	"context"
	"strings"

	"github.com/bytom/consensus"
	"github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc/types"
)

// action error
var (
	ErrBadActionType         = errors.New("bad action type")
	ErrBadAction             = errors.New("bad action object")
	ErrBadActionConstruction = errors.New("bad action construction")
)

// BuildRequest is main struct when building transactions
type BuildRequest struct {
	Tx        *types.TxData            `json:"base_transaction"`
	Actions   []map[string]interface{} `json:"actions"`
	TTL       json.Duration            `json:"ttl"`
	TimeRange uint64                   `json:"time_range"`
}

func (a *API) completeMissingIDs(ctx context.Context, br *BuildRequest) error {
	for i, m := range br.Actions {
		if err := a.completeMissingAssetID(m, i); err != nil {
			return err
		}
		if err := a.completeMissingAccountID(m, i, ctx); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) completeMissingAssetID(m map[string]interface{}, index int) error {
	id, _ := m["asset_id"].(string)
	alias, _ := m["asset_alias"].(string)
	if id == "" && alias != "" {
		alias = strings.ToUpper(alias)
		switch alias {
		case consensus.BTMAlias:
			m["asset_id"] = consensus.BTMAssetID.String()
		default:
			asset, err := a.wallet.AssetReg.FindByAlias(alias)
			if err != nil {
				return errors.WithDetailf(err, "invalid asset alias %s on action %d", alias, index)
			}
			m["asset_id"] = asset.AssetID.String()
		}
	}
	return nil
}

func (a *API) completeMissingAccountID(m map[string]interface{}, index int, ctx context.Context) error {
	id, _ := m["account_id"].(string)
	alias, _ := m["account_alias"].(string)
	if id == "" && alias != "" {
		acc, err := a.wallet.AccountMgr.FindByAlias(alias)
		if err != nil {
			return errors.WithDetailf(err, "invalid account alias %s on action %d", alias, index)
		}
		m["account_id"] = acc.ID
	}
	return nil
}
