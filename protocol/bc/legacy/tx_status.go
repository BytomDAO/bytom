package legacy

import "errors"

const (
	maxBitmapSize = 8388608
	bitsPerByte   = 8
)

var errOverRange = errors.New("bitmap range exceed the limit")
var errBadRange = errors.New("bitmap get a unexisted bit")

type TransactionStatus struct {
	bitmap []byte
}

func NewTransactionStatus() *TransactionStatus {
	return &TransactionStatus{
		bitmap: []byte{},
	}
}

func (ts *TransactionStatus) SetStatus(i int, gasOnly bool) error {
	if i >= maxBitmapSize {
		return errOverRange
	}

	index, pos := i/bitsPerByte, i%bitsPerByte
	for len(ts.bitmap) < index+1 {
		ts.bitmap = append(ts.bitmap, 0)
	}

	if gasOnly {
		ts.bitmap[index] |= 0x01 << uint8(pos)
	} else {
		ts.bitmap[index] &^= 0x01 << uint8(pos)
	}
	return nil
}

func (ts *TransactionStatus) GetStatus(i int) (bool, error) {
	if i >= maxBitmapSize {
		return false, errOverRange
	}

	index, pos := i/bitsPerByte, i%bitsPerByte
	for len(ts.bitmap) < index+1 {
		return false, errBadRange
	}

	result := (ts.bitmap[index] >> uint8(pos)) & 0x01
	return result == 1, nil
}
