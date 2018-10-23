package api

import (
	"context"
	"encoding/hex"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/crypto/ed25519/chainkd"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// POST /list-accounts
func (a *API) listAccounts(ctx context.Context, filter struct {
	ID    string `json:"id"`
	Alias string `json:"alias"`
}) Response {
	accountID := filter.ID
	if filter.Alias != "" {
		acc, err := a.wallet.AccountMgr.FindByAlias(filter.Alias)
		if err != nil {
			return NewErrorResponse(err)
		}
		accountID = acc.ID
	}

	accounts, err := a.wallet.AccountMgr.ListAccounts(accountID)
	if err != nil {
		log.Errorf("listAccounts: %v", err)
		return NewErrorResponse(err)
	}

	annotatedAccounts := []query.AnnotatedAccount{}
	for _, acc := range accounts {
		annotatedAccounts = append(annotatedAccounts, *account.Annotated(acc))
	}

	return NewSuccessResponse(annotatedAccounts)
}

// POST /get-asset
func (a *API) getAsset(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	asset, err := a.wallet.AssetReg.GetAsset(filter.ID)
	if err != nil {
		log.Errorf("getAsset: %v", err)
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(asset)
}

// POST /list-assets
func (a *API) listAssets(ctx context.Context, filter struct {
	ID string `json:"id"`
}) Response {
	assets, err := a.wallet.AssetReg.ListAssets(filter.ID)
	if err != nil {
		log.Errorf("listAssets: %v", err)
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(assets)
}

// POST /list-balances
func (a *API) listBalances(ctx context.Context, filter struct {
	AccountID    string `json:"account_id"`
	AccountAlias string `json:"account_alias"`
}) Response {
	accountID := filter.AccountID
	if filter.AccountAlias != "" {
		acc, err := a.wallet.AccountMgr.FindByAlias(filter.AccountAlias)
		if err != nil {
			return NewErrorResponse(err)
		}
		accountID = acc.ID
	}

	balances, err := a.wallet.GetAccountBalances(accountID, "")
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(balances)
}

// POST /get-transaction
func (a *API) getTransaction(ctx context.Context, txInfo struct {
	TxID string `json:"tx_id"`
}) Response {
	var annotatedTx *query.AnnotatedTx
	var err error

	annotatedTx, err = a.wallet.GetTransactionByTxID(txInfo.TxID)
	if err != nil {
		// transaction not found in blockchain db, search it from unconfirmed db
		annotatedTx, err = a.wallet.GetUnconfirmedTxByTxID(txInfo.TxID)
		if err != nil {
			return NewErrorResponse(err)
		}
	}

	return NewSuccessResponse(annotatedTx)
}

// POST /list-transactions
func (a *API) listTransactions(ctx context.Context, filter struct {
	ID          string `json:"id"`
	AccountID   string `json:"account_id"`
	Detail      bool   `json:"detail"`
	Unconfirmed bool   `json:"unconfirmed"`
	From        uint   `json:"from"`
	Count       uint   `json:"count"`
}) Response {
	transactions := []*query.AnnotatedTx{}
	var err error
	var transaction *query.AnnotatedTx

	if filter.ID != "" {
		transaction, err = a.wallet.GetTransactionByTxID(filter.ID)
		if err != nil && filter.Unconfirmed {
			transaction, err = a.wallet.GetUnconfirmedTxByTxID(filter.ID)
			if err != nil {
				return NewErrorResponse(err)
			}
		}
		transactions = []*query.AnnotatedTx{transaction}
	} else {
		transactions, err = a.wallet.GetTransactions(filter.AccountID)
		if err != nil {
			return NewErrorResponse(err)
		}

		if filter.Unconfirmed {
			unconfirmedTxs, err := a.wallet.GetUnconfirmedTxs(filter.AccountID)
			if err != nil {
				return NewErrorResponse(err)
			}
			transactions = append(unconfirmedTxs, transactions...)
		}
	}

	if filter.Detail == false {
		txSummary := a.wallet.GetTransactionsSummary(transactions)
		start, end := getPageRange(len(txSummary), filter.From, filter.Count)
		return NewSuccessResponse(txSummary[start:end])
	}
	start, end := getPageRange(len(transactions), filter.From, filter.Count)
	return NewSuccessResponse(transactions[start:end])
}

// POST /get-unconfirmed-transaction
func (a *API) getUnconfirmedTx(ctx context.Context, filter struct {
	TxID chainjson.HexBytes `json:"tx_id"`
}) Response {
	var tmpTxID [32]byte
	copy(tmpTxID[:], filter.TxID[:])

	txHash := bc.NewHash(tmpTxID)
	txPool := a.chain.GetTxPool()
	txDesc, err := txPool.GetTransaction(&txHash)
	if err != nil {
		return NewErrorResponse(err)
	}

	tx := &BlockTx{
		ID:         txDesc.Tx.ID,
		Version:    txDesc.Tx.Version,
		Size:       txDesc.Tx.SerializedSize,
		TimeRange:  txDesc.Tx.TimeRange,
		Inputs:     []*query.AnnotatedInput{},
		Outputs:    []*query.AnnotatedOutput{},
		StatusFail: txDesc.StatusFail,
	}

	resOutID := txDesc.Tx.ResultIds[0]
	resOut := txDesc.Tx.Entries[*resOutID]
	switch out := resOut.(type) {
	case *bc.Output:
		tx.MuxID = *out.Source.Ref
	case *bc.Retirement:
		tx.MuxID = *out.Source.Ref
	}

	for i := range txDesc.Tx.Inputs {
		tx.Inputs = append(tx.Inputs, a.wallet.BuildAnnotatedInput(txDesc.Tx, uint32(i)))
	}
	for i := range txDesc.Tx.Outputs {
		tx.Outputs = append(tx.Outputs, a.wallet.BuildAnnotatedOutput(txDesc.Tx, i))
	}

	return NewSuccessResponse(tx)
}

type unconfirmedTxsResp struct {
	Total uint64    `json:"total"`
	TxIDs []bc.Hash `json:"tx_ids"`
}

// POST /list-unconfirmed-transactions
func (a *API) listUnconfirmedTxs(ctx context.Context) Response {
	txIDs := []bc.Hash{}

	txPool := a.chain.GetTxPool()
	txs := txPool.GetTransactions()
	for _, txDesc := range txs {
		txIDs = append(txIDs, bc.Hash(txDesc.Tx.ID))
	}

	return NewSuccessResponse(&unconfirmedTxsResp{
		Total: uint64(len(txIDs)),
		TxIDs: txIDs,
	})
}

// RawTx is the tx struct for getRawTransaction
type RawTx struct {
	ID        bc.Hash                  `json:"tx_id"`
	Version   uint64                   `json:"version"`
	Size      uint64                   `json:"size"`
	TimeRange uint64                   `json:"time_range"`
	Inputs    []*query.AnnotatedInput  `json:"inputs"`
	Outputs   []*query.AnnotatedOutput `json:"outputs"`
	Fee       uint64                   `json:"fee"`
}

// POST /decode-raw-transaction
func (a *API) decodeRawTransaction(ctx context.Context, ins struct {
	Tx types.Tx `json:"raw_transaction"`
}) Response {
	tx := &RawTx{
		ID:        ins.Tx.ID,
		Version:   ins.Tx.Version,
		Size:      ins.Tx.SerializedSize,
		TimeRange: ins.Tx.TimeRange,
		Inputs:    []*query.AnnotatedInput{},
		Outputs:   []*query.AnnotatedOutput{},
	}

	for i := range ins.Tx.Inputs {
		tx.Inputs = append(tx.Inputs, a.wallet.BuildAnnotatedInput(&ins.Tx, uint32(i)))
	}
	for i := range ins.Tx.Outputs {
		tx.Outputs = append(tx.Outputs, a.wallet.BuildAnnotatedOutput(&ins.Tx, i))
	}

	tx.Fee = txbuilder.CalculateTxFee(&ins.Tx)
	return NewSuccessResponse(tx)
}

// POST /list-unspent-outputs
func (a *API) listUnspentOutputs(ctx context.Context, filter struct {
	AccountID     string `json:"account_id"`
	AccountAlias  string `json:"account_alias"`
	ID            string `json:"id"`
	Unconfirmed   bool   `json:"unconfirmed"`
	SmartContract bool   `json:"smart_contract"`
	From          uint   `json:"from"`
	Count         uint   `json:"count"`
}) Response {
	accountID := filter.AccountID
	if filter.AccountAlias != "" {
		acc, err := a.wallet.AccountMgr.FindByAlias(filter.AccountAlias)
		if err != nil {
			return NewErrorResponse(err)
		}
		accountID = acc.ID
	}
	accountUTXOs := a.wallet.GetAccountUtxos(accountID, filter.ID, filter.Unconfirmed, filter.SmartContract)

	UTXOs := []query.AnnotatedUTXO{}
	for _, utxo := range accountUTXOs {
		UTXOs = append([]query.AnnotatedUTXO{{
			AccountID:           utxo.AccountID,
			OutputID:            utxo.OutputID.String(),
			SourceID:            utxo.SourceID.String(),
			AssetID:             utxo.AssetID.String(),
			Amount:              utxo.Amount,
			SourcePos:           utxo.SourcePos,
			Program:             fmt.Sprintf("%x", utxo.ControlProgram),
			ControlProgramIndex: utxo.ControlProgramIndex,
			Address:             utxo.Address,
			ValidHeight:         utxo.ValidHeight,
			Alias:               a.wallet.AccountMgr.GetAliasByID(utxo.AccountID),
			AssetAlias:          a.wallet.AssetReg.GetAliasByID(utxo.AssetID.String()),
			Change:              utxo.Change,
		}}, UTXOs...)
	}
	start, end := getPageRange(len(UTXOs), filter.From, filter.Count)
	return NewSuccessResponse(UTXOs[start:end])
}

// return gasRate
func (a *API) gasRate() Response {
	gasrate := map[string]int64{"gas_rate": consensus.VMGasRate}
	return NewSuccessResponse(gasrate)
}

// PubKeyInfo is structure of pubkey info
type PubKeyInfo struct {
	Pubkey string               `json:"pubkey"`
	Path   []chainjson.HexBytes `json:"derivation_path"`
}

// AccountPubkey is detail of account pubkey info
type AccountPubkey struct {
	RootXPub    chainkd.XPub `json:"root_xpub"`
	PubKeyInfos []PubKeyInfo `json:"pubkey_infos"`
}

func getPubkey(account *account.Account, change bool, index uint64) (*ed25519.PublicKey, []chainjson.HexBytes, error) {
	rawPath, err := signers.Path(account.Signer, signers.AccountKeySpace, change, index)
	if err != nil {
		return nil, nil, err
	}
	derivedXPub := account.XPubs[0].Derive(rawPath)
	pubkey := derivedXPub.PublicKey()
	var path []chainjson.HexBytes
	for _, p := range rawPath {
		path = append(path, chainjson.HexBytes(p))
	}

	return &pubkey, path, nil
}

// POST /list-pubkeys
func (a *API) listPubKeys(ctx context.Context, ins struct {
	AccountID    string `json:"account_id"`
	AccountAlias string `json:"account_alias"`
	PublicKey    string `json:"public_key"`
}) Response {
	var err error
	account := &account.Account{}
	if ins.AccountAlias != "" {
		account, err = a.wallet.AccountMgr.FindByAlias(ins.AccountAlias)
	} else {
		account, err = a.wallet.AccountMgr.FindByID(ins.AccountID)
	}

	if err != nil {
		return NewErrorResponse(err)
	}

	pubKeyInfos := []PubKeyInfo{}
	if account.DeriveRule == signers.BIP0032 {
		idx := a.wallet.AccountMgr.GetContractIndex(account.ID)
		for i := uint64(1); i <= idx; i++ {
			pubkey, path, err := getPubkey(account, false, i)
			if err != nil {
				return NewErrorResponse(err)
			}
			if ins.PublicKey != "" && ins.PublicKey != hex.EncodeToString(*pubkey) {
				continue
			}
			pubKeyInfos = append(pubKeyInfos, PubKeyInfo{
				Pubkey: hex.EncodeToString(*pubkey),
				Path:   path,
			})
		}
	} else if account.DeriveRule == signers.BIP0044 {
		idx := a.wallet.AccountMgr.GetBip44ContractIndex(account.ID, true)
		for i := uint64(1); i <= idx; i++ {
			pubkey, path, err := getPubkey(account, true, i)
			if err != nil {
				return NewErrorResponse(err)
			}
			if ins.PublicKey != "" && ins.PublicKey != hex.EncodeToString(*pubkey) {
				continue
			}
			pubKeyInfos = append(pubKeyInfos, PubKeyInfo{
				Pubkey: hex.EncodeToString(*pubkey),
				Path:   path,
			})
		}

		idx = a.wallet.AccountMgr.GetBip44ContractIndex(account.ID, false)
		for i := uint64(1); i <= idx; i++ {
			pubkey, path, err := getPubkey(account, false, i)
			if err != nil {
				return NewErrorResponse(err)
			}
			if ins.PublicKey != "" && ins.PublicKey != hex.EncodeToString(*pubkey) {
				continue
			}
			pubKeyInfos = append(pubKeyInfos, PubKeyInfo{
				Pubkey: hex.EncodeToString(*pubkey),
				Path:   path,
			})
		}
	}

	if len(pubKeyInfos) == 0 {
		return NewErrorResponse(errors.New("Not found publickey for the account"))
	}

	return NewSuccessResponse(&AccountPubkey{
		RootXPub:    account.XPubs[0],
		PubKeyInfos: pubKeyInfos,
	})
}
