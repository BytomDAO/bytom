package account

import (
	"encoding/json"
	"fmt"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

// AnnotateTxs adds account data to transactions
func (m *Manager) AnnotateTxs(txs []*query.AnnotatedTx) error {

	outputIDs := make([][]byte, 0)
	inputs := make(map[bc.Hash]*query.AnnotatedInput)
	outputs := make(map[bc.Hash]*query.AnnotatedOutput)

	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.SpentOutputID == nil {
				continue
			}

			inputs[*in.SpentOutputID] = in
			outputIDs = append(outputIDs, in.SpentOutputID.Bytes())
		}
		for _, out := range tx.Outputs {
			if out.Type == "retire" {
				continue
			}

			outputs[out.OutputID] = out
			outputIDs = append(outputIDs, out.OutputID.Bytes())
		}
	}

	// Look up all of the spent and created outputs. If any of them are
	// account UTXOs add the account annotations to the inputs and outputs.

	accountUTXO := UTXO{}
	account := Account{}
	rawOutputID := new([32]byte)
	outputHash := bc.Hash{}

	for _, outputID := range outputIDs {

		accountUTXOValue := m.db.Get(accountUTXOKey(string(outputID)))
		if accountUTXOValue == nil {
			continue
		}
		if err := json.Unmarshal(accountUTXOValue, &accountUTXO); err != nil {
			return errors.Wrap(err)
		}
		copy(rawOutputID[:], accountUTXO.OutputID)
		outputHash = bc.NewHash(*rawOutputID)

		accountValue := m.db.Get(accountKey(accountUTXO.AccountID))
		if accountValue == nil {
			return errors.Wrap(fmt.Errorf("failed get account:%s ", accountUTXO.AccountID))
		}
		if err := json.Unmarshal(accountValue, &account); err != nil {
			return errors.Wrap(err)
		}

		aa, err := Annotated(&account)
		if err != nil {
			return errors.Wrap(err)
		}

		spendingInput, ok := inputs[outputHash]
		if ok {
			spendingInput.AccountID = aa.ID
			spendingInput.AccountAlias = aa.Alias
			spendingInput.AccountTags = aa.Tags
		}

		out, ok := outputs[outputHash]
		if ok {
			out.AccountID = aa.ID
			out.AccountAlias = aa.Alias
			out.AccountTags = aa.Tags

			if accountUTXO.Change {
				out.Purpose = "change"
			} else {
				out.Purpose = "receive"
			}
		}
	}

	return nil
}
