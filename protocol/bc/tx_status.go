package bc

import (
	"encoding/json"
	"io"
)

// WriteTo will write TxVerifyResult struct to io.Writer
func (tvr *TxVerifyResult) WriteTo(w io.Writer) (int64, error) {
	bytes, err := json.Marshal(tvr)
	if err != nil {
		return 0, err
	}

	n, err := w.Write(bytes)
	return int64(n), err
}
