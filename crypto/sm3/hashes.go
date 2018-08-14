package sm3

// Sum256 returns the SM3 digest of the data.
func Sum256(data []byte) (digest []byte) {
	h := New()
	h.Write(data)
	h.Sum(digest[:0])
	return
}
