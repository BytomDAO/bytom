package api

import (
	"github.com/bytom/errors"
)

var (
	errInvalidPageSize = errors.New("invalid page size")
	errInvalidCurrentPage = errors.New("invalid current page")
)

// Get the start and end of the page.
func getPageRange(total int, pageSize int, currentPage int) (int, int, error) {
	if pageSize == 0 && currentPage == 0 {
		return 0, total, nil
	}
	if pageSize <= 0 {
		return 0, 0, errInvalidPageSize
	}
	if currentPage <= 0 {
		return 0, 0, errInvalidCurrentPage
	}
	start := pageSize * (currentPage - 1)
	end := start + pageSize
	if start > total {start = total}
	if end > total {end = total}
	return start, end, nil
}