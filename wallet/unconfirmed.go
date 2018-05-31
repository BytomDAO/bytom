package wallet

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"fmt"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/protocol/bc/types"
)

const (
	//unconfirmedTxPrefix is txpool unconfirmed transactions prefix
	unconfirmedTxPrefix = "UTXS:"
)

func calcUnconfirmedKey(formatKey string) []byte {
	return []byte(unconfirmedTxPrefix + formatKey)
}

// SaveUnconfirmedTx save unconfirmed annotated transaction to the database
func (w *Wallet) SaveUnconfirmedTx(tx *types.Tx) error {
	annotatedTx := &query.AnnotatedTx{
		ID:      tx.ID,
		Inputs:  make([]*query.AnnotatedInput, 0, len(tx.Inputs)),
		Outputs: make([]*query.AnnotatedOutput, 0, len(tx.Outputs)),
		Size:    tx.SerializedSize,
	}

	for i := range tx.Inputs {
		annotatedTx.Inputs = append(annotatedTx.Inputs, w.BuildAnnotatedInput(tx, uint32(i)))
	}
	for i := range tx.Outputs {
		annotatedTx.Outputs = append(annotatedTx.Outputs, w.BuildAnnotatedOutput(tx, i))
	}

	// annotate account and asset
	annotatedTxs := []*query.AnnotatedTx{}
	annotatedTxs = append(annotatedTxs, annotatedTx)
	annotateTxsAccount(annotatedTxs, w.DB)
	annotateTxsAsset(w, annotatedTxs)

	rawTx, err := json.Marshal(annotatedTxs[0])
	if err != nil {
		log.WithField("err", err).Error("inserting unconfirmed annotated transaction to db")
		return err
	}

	w.DB.Set(calcUnconfirmedKey(tx.ID.String()), rawTx)
	return nil
}

// DeleteUnconfirmedTxs delete unconfirmed annotated transaction from the database
func (w *Wallet) DeleteUnconfirmedTxs(txIDs []string) {
	for _, txID := range txIDs {
		if exist := w.DB.Get(calcUnconfirmedKey(txID)); exist != nil {
			w.DB.Delete(calcUnconfirmedKey(txID))
		}
	}
}

// RescanWalletTxPool rescan txPool
func (w *Wallet) RescanWalletTxPool() []string {
	txIDs := []string{}

	txPool := w.chain.GetTxPool()
	txs := txPool.GetTransactions()
	for _, txDesc := range txs {
		txIDs = append(txIDs, txDesc.Tx.ID.String())
	}

	return txIDs
}

// GetUnconfirmedTxByTxID get unconfirmed transaction by txID
func (w *Wallet) GetUnconfirmedTxByTxID(txID string) (*query.AnnotatedTx, error) {
	formatKey := w.DB.Get(calcUnconfirmedKey(txID))
	if formatKey == nil {
		return nil, fmt.Errorf("Not found unconfirmed transaction(tx_id=%s) ", txID)
	}

	annotatedTx := &query.AnnotatedTx{}
	txInfo := w.DB.Get(calcAnnotatedKey(string(formatKey)))
	if err := json.Unmarshal(txInfo, annotatedTx); err != nil {
		return nil, err
	}

	return annotatedTx, nil
}

// GetUnconfirmedTxsByAccountID get account unconfirmed txs by account ID
func (w *Wallet) GetUnconfirmedTxsByAccountID(accountID string) ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}

	txIter := w.DB.IteratorPrefix([]byte(unconfirmedTxPrefix))
	defer txIter.Release()
	for txIter.Next() {
		annotatedTx := &query.AnnotatedTx{}
		if err := json.Unmarshal(txIter.Value(), &annotatedTx); err != nil {
			return nil, err
		}

		if accountID == "" {
			annotatedTxs = append(annotatedTxs, annotatedTx)
			continue
		}

		if findTransactionsByAccount(annotatedTx, accountID) {
			annotatedTxs = append(annotatedTxs, annotatedTx)
		}
	}

	return annotatedTxs, nil
}
