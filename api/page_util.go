package api

import (
	"github.com/bytom/errors"
)

var (
	errInvalidFromPosition = errors.New("invalid from position")
	errInvalidCountNum = errors.New("invalid count num")
)

// Get the start and end of the page.
func getPageRange(total int, from int, count int) (int, int, error) {
	if from == 0 && count == 0 {
		return 0, total, nil
	}
	if from < 0 {
		return 0, 0, errInvalidFromPosition
	}
	if count <= 0 {
		return 0, 0, errInvalidCountNum
	}
	start := from
	end := from + count
	if start > total {start = total}
	if end > total {end = total}
	return start, end, nil
}