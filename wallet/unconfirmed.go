package wallet

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/protocol/bc/types"
	"sort"
)

const (
	//UnconfirmedTxPrefix is txpool unconfirmed transactions prefix
	UnconfirmedTxPrefix = "UTXS:"
)

func calcUnconfirmedTxKey(formatKey string) []byte {
	return []byte(UnconfirmedTxPrefix + formatKey)
}

// SaveUnconfirmedTx save unconfirmed annotated transaction to the database
func (w *Wallet) SaveUnconfirmedTx(tx *types.Tx) error {
	annotatedTx := &query.AnnotatedTx{
		ID:        tx.ID,
		Timestamp: uint64(time.Now().Unix()),
		Inputs:    make([]*query.AnnotatedInput, 0, len(tx.Inputs)),
		Outputs:   make([]*query.AnnotatedOutput, 0, len(tx.Outputs)),
		Size:      tx.SerializedSize,
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

	rawTx, err := json.Marshal(annotatedTxs[0])
	if err != nil {
		return err
	}

	w.DB.Set(calcUnconfirmedTxKey(tx.ID.String()), rawTx)
	return nil
}

// GetUnconfirmedTxByTxID get unconfirmed transaction by txID
func (w *Wallet) GetUnconfirmedTxByTxID(txID string) (*query.AnnotatedTx, error) {
	annotatedTx := &query.AnnotatedTx{}
	txInfo := w.DB.Get(calcUnconfirmedTxKey(txID))
	if txInfo == nil {
		return nil, fmt.Errorf("No transaction(tx_id=%s) from txpool", txID)
	}

	if err := json.Unmarshal(txInfo, annotatedTx); err != nil {
		return nil, err
	}
	annotateTxsAsset(w, []*query.AnnotatedTx{annotatedTx})

	return annotatedTx, nil
}

// SortByTimestamp implements sort.Interface for AnnotatedTx slices
type SortByTimestamp []*query.AnnotatedTx

func (a SortByTimestamp) Len() int           { return len(a) }
func (a SortByTimestamp) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortByTimestamp) Less(i, j int) bool { return a[i].Timestamp > a[j].Timestamp }

// GetUnconfirmedTxs get account unconfirmed transactions, filter transactions by accountID when accountID is not empty
func (w *Wallet) GetUnconfirmedTxs(accountID string) ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}

	txIter := w.DB.IteratorPrefix([]byte(UnconfirmedTxPrefix))
	defer txIter.Release()
	for txIter.Next() {
		annotatedTx := &query.AnnotatedTx{}
		if err := json.Unmarshal(txIter.Value(), &annotatedTx); err != nil {
			return nil, err
		}

		if accountID == "" || findTransactionsByAccount(annotatedTx, accountID) {
			annotateTxsAsset(w, []*query.AnnotatedTx{annotatedTx})
			annotatedTxs = append([]*query.AnnotatedTx{annotatedTx}, annotatedTxs...)
		}
	}

	// sort SortByTimestamp by timestamp
	sort.Sort(SortByTimestamp(annotatedTxs))
	return annotatedTxs, nil
}
