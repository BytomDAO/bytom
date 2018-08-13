package api

// Get the start and end of the page.
func getPageRange(size int, from uint, count uint) (uint, uint) {
	total := uint(size)
	if from == 0 && count == 0 {
		return 0, total
	}
	start := from
	end := from + count
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	return start, end
}
