package wallet

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/sha3pool"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/vm/vmutil"
)

// annotateTxs adds asset data to transactions
func annotateTxsAsset(w *Wallet, txs []*query.AnnotatedTx) {
	for i, tx := range txs {
		for j, input := range tx.Inputs {
			txs[i].Inputs[j].AssetAlias, txs[i].Inputs[j].AssetDefinition =
				w.getAliasDefinition(input.AssetID)
		}
		for k, output := range tx.Outputs {
			txs[i].Outputs[k].AssetAlias, txs[i].Outputs[k].AssetDefinition =
				w.getAliasDefinition(output.AssetID)
		}
	}
}

func (w *Wallet) getExternalDefinition(assetID *bc.AssetID) *chainjson.HexBytes {

	definitionByte := w.DB.Get(asset.CalcExtAssetKey(assetID))
	if definitionByte == nil {
		return nil
	}

	definitionMap := make(map[string]interface{})
	if err := json.Unmarshal(definitionByte, &definitionMap); err != nil {
		return nil
	}

	saveAlias := assetID.String()
	storeBatch := w.DB.NewBatch()

	externalAsset := &asset.Asset{AssetID: *assetID, Alias: &saveAlias, DefinitionMap: definitionMap, Signer: &signers.Signer{Type: "external"}}
	if rawAsset, err := json.Marshal(externalAsset); err == nil {
		log.WithFields(log.Fields{"assetID": assetID.String(), "alias": saveAlias}).Info("index external asset")
		storeBatch.Set(asset.Key(assetID), rawAsset)
	}
	storeBatch.Set(asset.AliasKey(saveAlias), []byte(assetID.String()))
	storeBatch.Write()

	d := chainjson.HexBytes(definitionByte)
	return &d

}

func (w *Wallet) getAliasDefinition(assetID bc.AssetID) (string, *chainjson.HexBytes) {
	//btm
	if assetID.String() == consensus.BTMAssetID.String() {
		alias := consensus.BTMAlias
		definition := &asset.DefaultNativeAsset.RawDefinitionByte

		return alias, definition
	}

	//local asset and saved external asset
	if localAsset, err := w.AssetReg.FindByID(nil, &assetID); err == nil {
		alias := *localAsset.Alias
		definition := &localAsset.RawDefinitionByte
		return alias, definition
	}

	//external asset
	if definition := w.getExternalDefinition(&assetID); definition != nil {
		return assetID.String(), definition
	}

	return "", nil
}

// annotateTxs adds account data to transactions
func annotateTxsAccount(txs []*query.AnnotatedTx, walletDB db.DB) {
	for i, tx := range txs {
		for j, input := range tx.Inputs {
			//issue asset tx input SpentOutputID is nil
			if input.SpentOutputID == nil {
				continue
			}
			localAccount, err := getAccountFromUTXO(*input.SpentOutputID, walletDB)
			if localAccount == nil || err != nil {
				continue
			}
			txs[i].Inputs[j].AccountAlias = localAccount.Alias
			txs[i].Inputs[j].AccountID = localAccount.ID
		}
		for j, output := range tx.Outputs {
			localAccount, err := getAccountFromACP(output.ControlProgram, walletDB)
			if localAccount == nil || err != nil {
				continue
			}
			txs[i].Outputs[j].AccountAlias = localAccount.Alias
			txs[i].Outputs[j].AccountID = localAccount.ID
		}
	}
}

func getAccountFromUTXO(outputID bc.Hash, walletDB db.DB) (*account.Account, error) {
	accountUTXO := account.UTXO{}
	localAccount := account.Account{}

	accountUTXOValue := walletDB.Get(account.StandardUTXOKey(outputID))
	if accountUTXOValue == nil {
		return nil, fmt.Errorf("failed get account utxo:%x ", outputID)
	}

	if err := json.Unmarshal(accountUTXOValue, &accountUTXO); err != nil {
		return nil, err
	}

	accountValue := walletDB.Get(account.Key(accountUTXO.AccountID))
	if accountValue == nil {
		return nil, fmt.Errorf("failed get account:%s ", accountUTXO.AccountID)
	}
	if err := json.Unmarshal(accountValue, &localAccount); err != nil {
		return nil, err
	}

	return &localAccount, nil
}

func getAccountFromACP(program []byte, walletDB db.DB) (*account.Account, error) {
	var hash common.Hash
	accountCP := account.CtrlProgram{}
	localAccount := account.Account{}

	sha3pool.Sum256(hash[:], program)

	rawProgram := walletDB.Get(account.CPKey(hash))
	if rawProgram == nil {
		return nil, fmt.Errorf("failed get account control program:%x ", hash)
	}

	if err := json.Unmarshal(rawProgram, &accountCP); err != nil {
		return nil, err
	}

	accountValue := walletDB.Get(account.Key(accountCP.AccountID))
	if accountValue == nil {
		return nil, fmt.Errorf("failed get account:%s ", accountCP.AccountID)
	}

	if err := json.Unmarshal(accountValue, &localAccount); err != nil {
		return nil, err
	}

	return &localAccount, nil
}

var emptyJSONObject = chainjson.HexBytes(`{}`)

func isValidJSON(b []byte) bool {
	var v interface{}
	err := json.Unmarshal(b, &v)
	return err == nil
}

func buildAnnotatedTransaction(orig *legacy.Tx, b *legacy.Block, statusFail bool, indexInBlock int) *query.AnnotatedTx {
	tx := &query.AnnotatedTx{
		ID:                     orig.ID,
		Timestamp:              b.Time(),
		BlockID:                b.Hash(),
		BlockHeight:            b.Height,
		Position:               uint32(indexInBlock),
		BlockTransactionsCount: uint32(len(b.Transactions)),
		Inputs:                 make([]*query.AnnotatedInput, 0, len(orig.Inputs)),
		Outputs:                make([]*query.AnnotatedOutput, 0, len(orig.Outputs)),
		StatusFail:             statusFail,
	}
	for i := range orig.Inputs {
		tx.Inputs = append(tx.Inputs, buildAnnotatedInput(orig, uint32(i)))
	}
	for i := range orig.Outputs {
		tx.Outputs = append(tx.Outputs, buildAnnotatedOutput(orig, i))
	}
	return tx
}

func buildAnnotatedInput(tx *legacy.Tx, i uint32) *query.AnnotatedInput {
	orig := tx.Inputs[i]
	in := &query.AnnotatedInput{
		AssetDefinition: &emptyJSONObject,
	}
	if !orig.IsCoinbase() {
		in.AssetID = orig.AssetID()
		in.Amount = orig.Amount()
	}

	id := tx.Tx.InputIDs[i]
	e := tx.Entries[id]
	switch e := e.(type) {
	case *bc.Spend:
		in.Type = "spend"
		in.ControlProgram = orig.ControlProgram()
		in.SpentOutputID = e.SpentOutputId
	case *bc.Issuance:
		in.Type = "issue"
		in.IssuanceProgram = orig.IssuanceProgram()
	case *bc.Coinbase:
		in.Type = "coinbase"
		in.Arbitrary = e.Arbitrary
	}
	return in
}

func buildAnnotatedOutput(tx *legacy.Tx, idx int) *query.AnnotatedOutput {
	orig := tx.Outputs[idx]
	outid := tx.OutputID(idx)
	out := &query.AnnotatedOutput{
		OutputID:        *outid,
		Position:        idx,
		AssetID:         *orig.AssetId,
		AssetDefinition: &emptyJSONObject,
		Amount:          orig.Amount,
		ControlProgram:  orig.ControlProgram,
	}
	if vmutil.IsUnspendable(out.ControlProgram) {
		out.Type = "retire"
	} else {
		out.Type = "control"
	}
	return out
}
