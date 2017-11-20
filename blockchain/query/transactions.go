package query

import (
	"fmt"
	"math"

	"github.com/bytom/blockchain/query/filter"
	"github.com/bytom/errors"
)

var filterTable = filter.Table{
	Name:  "annotated_txs",
	Alias: "txs",
	Columns: map[string]*filter.Column{
		"asset_id":           {Name: "assetid", Type: filter.String},
		"amount_lower_limit": {Name: "amountlower", Type: filter.Integer},
		"amount_upper_limit": {Name: "amountupper", Type: filter.Integer},
		"trans_type":         {Name: "transtype", Type: filter.String},
	},
}

var (
	//ErrBadAfter means malformed pagination parameter.
	ErrBadAfter = errors.New("malformed pagination parameter after")
	//ErrParameterCountMismatch means wrong number of parameters to query.
	ErrParameterCountMismatch = errors.New("wrong number of parameters to query")
)

//TxAfter means the last query block by a list-transactions query.
type TxAfter struct {
	// FromBlockHeight and FromPosition uniquely identify the last transaction returned
	// by a list-transactions query.
	//
	// If list-transactions is called with a time range instead of an `after`, these fields
	// are populated with the position of the transaction at the start of the time range.
	FromBlockHeight uint64 // exclusive
	FromPosition    uint32 // exclusive

	// StopBlockHeight identifies the last block that should be included in a transaction
	// list. It is used when list-transactions is called with a time range instead
	// of an `after`.
	StopBlockHeight uint64 // inclusive
}

func (after TxAfter) String() string {
	return fmt.Sprintf("%d:%d-%d", after.FromBlockHeight, after.FromPosition, after.StopBlockHeight)
}

//DecodeTxAfter decode tx from the last block.
func DecodeTxAfter(str string) (c TxAfter, err error) {
	var from, pos, stop uint64
	_, err = fmt.Sscanf(str, "%d:%d-%d", &from, &pos, &stop)
	if err != nil {
		return c, errors.Sub(ErrBadAfter, err)
	}
	if from > math.MaxInt64 ||
		pos > math.MaxUint32 ||
		stop > math.MaxInt64 {
		return c, errors.Wrap(ErrBadAfter)
	}
	return TxAfter{FromBlockHeight: from, FromPosition: uint32(pos), StopBlockHeight: stop}, nil
}

//ValidateTransactionFilter verify txfeed filter validity.
func ValidateTransactionFilter(filt string) error {
	_, err := filter.Parse(filt, &filterTable, nil)
	return err
}
