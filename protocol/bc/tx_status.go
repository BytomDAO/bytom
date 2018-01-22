package bc

import "errors"

const (
	maxBitmapSize = 8388608
	bitsPerByte   = 8
)

var errOverRange = errors.New("bitmap range exceed the limit")
var errBadRange = errors.New("bitmap get a unexisted bit")

func NewTransactionStatus() *TransactionStatus {
	return &TransactionStatus{
		Bitmap: []byte{0},
	}
}

func (ts *TransactionStatus) SetStatus(i int, gasOnly bool) error {
	if i >= maxBitmapSize {
		return errOverRange
	}

	index, pos := i/bitsPerByte, i%bitsPerByte
	for len(ts.Bitmap) < index+1 {
		ts.Bitmap = append(ts.Bitmap, 0)
	}

	if gasOnly {
		ts.Bitmap[index] |= 0x01 << uint8(pos)
	} else {
		ts.Bitmap[index] &^= 0x01 << uint8(pos)
	}
	return nil
}

func (ts *TransactionStatus) GetStatus(i int) (bool, error) {
	if i >= maxBitmapSize {
		return false, errOverRange
	}

	index, pos := i/bitsPerByte, i%bitsPerByte
	for len(ts.Bitmap) < index+1 {
		return false, errBadRange
	}

	result := (ts.Bitmap[index] >> uint8(pos)) & 0x01
	return result == 1, nil
}
