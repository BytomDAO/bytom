package wallet

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

// annotateTxs adds asset data to transactions
func annotateTxsAsset(txs []*query.AnnotatedTx, walletDB db.DB) {
	for i, tx := range txs {
		for j, input := range tx.Inputs {
			localAsset, err := getAliasFromAssetID(input.AssetID, walletDB)
			if localAsset == nil || err != nil {
				continue
			}
			txs[i].Inputs[j].AssetAlias = *localAsset.Alias
		}
		for j, output := range tx.Outputs {
			localAsset, err := getAliasFromAssetID(output.AssetID, walletDB)
			if localAsset == nil || err != nil {
				continue
			}
			txs[i].Outputs[j].AssetAlias = *localAsset.Alias
		}
	}
}

func getAliasFromAssetID(assetID bc.AssetID, walletDB db.DB) (*asset.Asset, error) {
	var localAsset asset.Asset
	rawAsset := walletDB.Get(asset.AssetKey(assetID.String()))
	if rawAsset == nil {
		return nil, nil
	}

	if err := json.Unmarshal(rawAsset, &localAsset); err != nil {
		log.WithFields(log.Fields{"warn": err, "asset id": assetID.String()}).Warn("look up asset")
		return nil, err
	}

	return &localAsset, nil
}

// annotateTxs adds account data to transactions
func annotateTxsAccount(txs []*query.AnnotatedTx, walletDB db.DB) {
	for i, tx := range txs {
		for j, input := range tx.Inputs {
			account, err := getAccountFromUTXO(*input.SpentOutputID, walletDB)
			if account == nil || err != nil {
				continue
			}
			txs[i].Inputs[j].AccountAlias = account.Alias
		}
		for j, output := range tx.Outputs {
			account, err := getAccountFromACP(output.ControlProgram, walletDB)
			if account == nil || err != nil {
				continue
			}
			txs[i].Outputs[j].AccountAlias = account.Alias
		}
	}
}

func getAccountFromUTXO(outputID bc.Hash, walletDB db.DB) (*account.Account, error) {
	accountUTXO := account.UTXO{}
	localAccount := account.Account{}

	accountUTXOValue := walletDB.Get(account.AccountUTXOKey(string(outputID.Bytes())))
	if accountUTXOValue == nil {
		return nil, errors.Wrap(fmt.Errorf("failed get account utxo:%x ", outputID))
	}

	if err := json.Unmarshal(accountUTXOValue, &accountUTXO); err != nil {
		return nil, errors.Wrap(err)
	}

	accountValue := walletDB.Get(account.AccountKey(accountUTXO.AccountID))
	if accountValue == nil {
		return nil, errors.Wrap(fmt.Errorf("failed get account:%s ", accountUTXO.AccountID))
	}
	if err := json.Unmarshal(accountValue, &localAccount); err != nil {
		return nil, errors.Wrap(err)
	}

	return &localAccount, nil
}

func getAccountFromACP(program []byte, walletDB db.DB) (*account.Account, error) {
	var hash [32]byte
	accountCP := account.CtrlProgram{}
	localAccount := account.Account{}

	sha3pool.Sum256(hash[:], program)

	rawProgram := walletDB.Get(account.AccountCPKey(hash))
	if rawProgram == nil {
		return nil, errors.Wrap(fmt.Errorf("failed get account control program:%x ", hash))
	}

	if err := json.Unmarshal(rawProgram, &accountCP); err != nil {
		return nil, errors.Wrap(err)
	}

	accountValue := walletDB.Get(account.AccountKey(accountCP.AccountID))
	if accountValue == nil {
		return nil, errors.Wrap(fmt.Errorf("failed get account:%s ", accountCP.AccountID))
	}

	if err := json.Unmarshal(accountValue, &localAccount); err != nil {
		return nil, errors.Wrap(err)
	}

	return &localAccount, nil
}
