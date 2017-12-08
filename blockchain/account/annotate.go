package account

import (
	"encoding/json"
	"fmt"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/tendermint/tmlibs/db"
	"github.com/bytom/crypto/sha3pool"
)

// AnnotateTxs adds account data to transactions
func AnnotateTxs(txs []*query.AnnotatedTx,walletDB db.DB) error {

	outputIDs := make([]bc.Hash, 0)
	inputs := make(map[bc.Hash]*query.AnnotatedInput)
	outputs := make(map[bc.Hash]*query.AnnotatedOutput)

	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.SpentOutputID == nil {
				continue
			}

			inputs[*in.SpentOutputID] = in
			outputIDs = append(outputIDs, *in.SpentOutputID)
		}
		for _, out := range tx.Outputs {
			if out.Type == "retire" {
				continue
			}

			outputs[out.OutputID] = out
			outputIDs = append(outputIDs, out.OutputID)
		}
	}

	// Look up all of the spent and created outputs. If any of them are
	// account UTXOs add the account annotations to the inputs and outputs.

	for _, outputID := range outputIDs {
		var account *Account
		var err error

		if account,err = getAccountFromUTXO(outputID,walletDB);err != nil{
			if account,err = getAccountFromACP(outputs[outputID].ControlProgram,walletDB);err != nil {
				return errors.Wrap(err)
			}
		}

		aa, err := Annotated(account)
		if err != nil {
			return errors.Wrap(err)
		}

		spendingInput, ok := inputs[outputID]
		if ok {
			spendingInput.AccountID = aa.ID
			spendingInput.AccountAlias = aa.Alias
			spendingInput.AccountTags = aa.Tags
		}

		out, ok := outputs[outputID]
		if ok {
			out.AccountID = aa.ID
			out.AccountAlias = aa.Alias
			out.AccountTags = aa.Tags
		}
	}

	return nil
}

func getAccountFromUTXO(outputID bc.Hash,walletDB db.DB) (*Account,error) {
	accountUTXO := UTXO{}
	account := Account{}

	accountUTXOValue := walletDB.Get(AccountUTXOKey(string(outputID.Bytes())))
	if accountUTXOValue == nil {
		return nil,errors.Wrap(fmt.Errorf("failed get account utxo:%x ", outputID))
	}

	if err := json.Unmarshal(accountUTXOValue, &accountUTXO); err != nil {
		return nil,errors.Wrap(err)
	}

	accountValue := walletDB.Get(AccountKey(accountUTXO.AccountID))
	if accountValue == nil {
		return nil,errors.Wrap(fmt.Errorf("failed get account:%s ", accountUTXO.AccountID))
	}
	if err := json.Unmarshal(accountValue, &account); err != nil {
		return nil,errors.Wrap(err)
	}

	return &account,nil
}

func getAccountFromACP(program []byte,walletDB db.DB) (*Account,error) {
	var hash [32]byte
	accountCP := CtrlProgram{}
	account := Account{}

	sha3pool.Sum256(hash[:], program)

	rawProgram := walletDB.Get(AccountCPKey(hash))
	if rawProgram == nil {
		return nil,errors.Wrap(fmt.Errorf("failed get account control program:%x ", hash))
	}

	if err := json.Unmarshal(rawProgram, &accountCP); err != nil {
		return nil,errors.Wrap(err)
	}

	accountValue := walletDB.Get(AccountKey(accountCP.AccountID))
	if accountValue == nil {
		return nil,errors.Wrap(fmt.Errorf("failed get account:%s ", accountCP.AccountID))
	}
	if err := json.Unmarshal(accountValue, &account); err != nil {
		return nil,errors.Wrap(err)
	}

	return &account,nil
}