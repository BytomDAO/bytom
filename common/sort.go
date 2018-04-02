package common

// timeSorter implements sort.Interface to allow a slice of timestamps to
// be sorted.
type TimeSorter []uint64

// Len returns the number of timestamps in the slice.  It is part of the
// sort.Interface implementation.
func (s TimeSorter) Len() int {
	return len(s)
}

// Swap swaps the timestamps at the passed indices.  It is part of the
// sort.Interface implementation.
func (s TimeSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less returns whether the timstamp with index i should sort before the
// timestamp with index j.  It is part of the sort.Interface implementation.
func (s TimeSorter) Less(i, j int) bool {
	return s[i] < s[j]
}
