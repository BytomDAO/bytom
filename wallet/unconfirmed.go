package wallet

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc/types"
)

const (
	//unconfirmedTxPrefix is txpool unconfirmed transactions prefix
	unconfirmedTxPrefix = "UTXS:"
)

func calcUnconfirmedTxKey(formatKey string) []byte {
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
		return err
	}

	w.DB.Set(calcUnconfirmedTxKey(tx.ID.String()), rawTx)
	log.Debugf("insert unconfirmed tx=%s into db", tx.ID.String())
	return nil
}

// GetUnconfirmedTxByTxID get unconfirmed transaction by txID
func (w *Wallet) GetUnconfirmedTxByTxID(txID string) (*query.AnnotatedTx, error) {
	annotatedTx := &query.AnnotatedTx{}
	txInfo := w.DB.Get(calcUnconfirmedTxKey(txID))
	if txInfo == nil {
		return nil, errors.WithData(ErrNotFoundTx, "not found tx=%s from txpool", txID)
	}

	if err := json.Unmarshal(txInfo, annotatedTx); err != nil {
		return nil, err
	}

	return annotatedTx, nil
}

// GetUnconfirmedTxs get account unconfirmed transactions, filter transactions by accountID when accountID is not empty
func (w *Wallet) GetUnconfirmedTxs(accountID string) ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}
	annotatedAccTxs := []*query.AnnotatedTx{}

	txIter := w.DB.IteratorPrefix([]byte(unconfirmedTxPrefix))
	defer txIter.Release()
	for txIter.Next() {
		annotatedTx := &query.AnnotatedTx{}
		if err := json.Unmarshal(txIter.Value(), &annotatedTx); err != nil {
			return nil, err
		}

		annotatedTxs = append(annotatedTxs, annotatedTx)
		if accountID != "" && findTransactionsByAccount(annotatedTx, accountID) {
			annotatedAccTxs = append(annotatedAccTxs, annotatedTx)
		}
	}

	if accountID != "" {
		return annotatedAccTxs, nil
	}
	return annotatedTxs, nil
}
