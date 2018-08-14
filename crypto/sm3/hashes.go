package sm3

// Sum256 returns the SM3 digest of the data.
func Sum256(data []byte) (digest []byte) {
	h := New()
	h.Write(data)
	h.Sum(digest[:0])
	return
}

// // Sum256 returns the SHA3-256 digest of the data.
// func Sum256(data []byte) (digest [32]byte) {
// 	h := New256()
// 	h.Write(data)
// 	h.Sum(digest[:0])
// 	return
// }
