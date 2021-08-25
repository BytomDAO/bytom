package server

import (
	"github.com/bytom/bytom/errors"
)

var (
	errMissingFilterKey  = errors.New("missing filter key")
	errInvalidFilterType = errors.New("invalid filter type")
)

// Display defines how the data is displayed
type Display struct {
	Filter map[string]interface{} `json:"filter"`
	Sorter Sorter                 `json:"sort"`
}

type Sorter struct {
	By    string `json:"by"`
	Order string `json:"order"`
}

// GetFilterString give the filter keyword return the string value
func (d *Display) GetFilterString(filterKey string) (string, error) {
	if _, ok := d.Filter[filterKey]; !ok {
		return "", errMissingFilterKey
	}
	switch val := d.Filter[filterKey].(type) {
	case string:
		return val, nil
	}
	return "", errInvalidFilterType
}

// GetFilterNum give the filter keyword return the numeric value
func (d *Display) GetFilterNum(filterKey string) (interface{}, error) {
	if _, ok := d.Filter[filterKey]; !ok {
		return 0, errMissingFilterKey
	}
	switch val := d.Filter[filterKey].(type) {
	case int, int16, int32, int64, int8, uint, uint16, uint32, uint64, uint8, float32, float64:
		return val, nil
	}

	return 0, errInvalidFilterType
}

// GetFilterBoolean give the filter keyword return the boolean value
func (d *Display) GetFilterBoolean(filterKey string) (bool, error) {
	if _, ok := d.Filter[filterKey]; !ok {
		return false, errMissingFilterKey
	}
	switch val := d.Filter[filterKey].(type) {
	case bool:
		return val, nil
	}
	return false, errInvalidFilterType
}
