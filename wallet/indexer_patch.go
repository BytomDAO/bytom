package wallet

import (
	"sort"

	"github.com/bytom/bytom/account"
)

// GetAddressBalances return appointed address all assets  balances
func (w *Wallet) GetAddressBalances(address string) ([]AddressBalance, error) {
	return w.indexBalancesEx(w.GetAddressUTXOs(address, false, false))
}

type Key struct {
	Address string
	Asset   string
}

// AddressBalance address balance
type AddressBalance struct {
	Address         string                 `json:"address"`
	AssetAlias      string                 `json:"asset_alias"`
	AssetID         string                 `json:"asset_id"`
	Amount          uint64                 `json:"amount"`
	AssetDefinition map[string]interface{} `json:"asset_definition"`
}

func (w *Wallet) indexBalancesEx(accountUTXOs []*account.UTXO) ([]AddressBalance, error) {
	address2balance := make(map[*Key]uint64)
	for _, utxo := range accountUTXOs {
		key := &Key{Address: utxo.Address, Asset: utxo.AssetID.String()}
		address2balance[key] += utxo.Amount
	}

	balances := []AddressBalance{}
	for _, key := range sortKeys(address2balance) {
		targetAsset, err := w.AssetReg.GetAsset(key.Asset)
		if err != nil {
			return nil, err
		}

		assetAlias := *targetAsset.Alias
		balances = append(balances, AddressBalance{
			Address:         key.Address,
			AssetID:         key.Asset,
			AssetAlias:      assetAlias,
			Amount:          address2balance[key],
			AssetDefinition: targetAsset.DefinitionMap,
		})
	}

	return balances, nil
}

func sortKeys(address2balance map[*Key]uint64) []*Key {
	var sortedKeys []*Key
	for key := range address2balance {
		sortedKeys = append(sortedKeys, key)
	}

	sort.Slice(sortedKeys, func(i, j int) bool {
		if sortedKeys[i].Address > sortedKeys[j].Address {
			return true
		}

		if (sortedKeys[i].Address == sortedKeys[j].Address) && (sortedKeys[i].Asset > sortedKeys[j].Asset) {
			return true
		}

		return false
	})

	return sortedKeys
}
