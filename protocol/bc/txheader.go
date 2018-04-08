package bc

import "io"

// TxHeader contains header information for a transaction. Every
// transaction on a blockchain contains exactly one TxHeader. The ID
// of the TxHeader is the ID of the transaction. TxHeader satisfies
// the Entry interface.

func (TxHeader) typ() string { return "txheader" }
func (h *TxHeader) writeForHash(w io.Writer) {
	mustWriteForHash(w, h.Version)
	mustWriteForHash(w, h.TimeRange)
	mustWriteForHash(w, h.ResultIds)
}

// NewTxHeader creates an new TxHeader.
func NewTxHeader(version, serializedSize, timeRange uint64, resultIDs []*Hash) *TxHeader {
	return &TxHeader{
		Version:        version,
		SerializedSize: serializedSize,
		TimeRange:      timeRange,
		ResultIds:      resultIDs,
	}
}
