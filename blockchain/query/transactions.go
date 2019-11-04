package query

import (
	"github.com/bytom/bytom/blockchain/query/filter"
	"github.com/bytom/bytom/errors"
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

//ValidateTransactionFilter verify txfeed filter validity.
func ValidateTransactionFilter(filt string) error {
	_, err := filter.Parse(filt, &filterTable, nil)
	return err
}
