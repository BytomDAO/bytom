package bc

import (
	"encoding/json"
	"errors"
	"io"
)

const transactionStatusVersion = 1

// NewTransactionStatus create a empty TransactionStatus struct
func NewTransactionStatus() *TransactionStatus {
	return &TransactionStatus{
		Version:      transactionStatusVersion,
		VerifyStatus: []*TxVerifyResult{},
	}
}

// SetStatus set the tx status of given index
func (ts *TransactionStatus) SetStatus(i int, gasOnly bool) error {
	if i > len(ts.VerifyStatus) {
		return errors.New("setStatus should be set one by one")
	}

	if i == len(ts.VerifyStatus) {
		ts.VerifyStatus = append(ts.VerifyStatus, &TxVerifyResult{StatusFail: gasOnly})
	} else {
		ts.VerifyStatus[i].StatusFail = gasOnly
	}
	return nil
}

// GetStatus get the tx status of given index
func (ts *TransactionStatus) GetStatus(i int) (bool, error) {
	if i >= len(ts.VerifyStatus) {
		return false, errors.New("GetStatus is out of range")
	}

	return ts.VerifyStatus[i].StatusFail, nil
}

// WriteTo will write TxVerifyResult struct to io.Writer
func (tvr *TxVerifyResult) WriteTo(w io.Writer) (int64, error) {
	bytes, err := json.Marshal(tvr)
	if err != nil {
		return 0, err
	}

	n, err := w.Write(bytes)
	return int64(n), err
}
