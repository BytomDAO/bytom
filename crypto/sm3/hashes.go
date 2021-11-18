package sm3

// Sum256 returns the SM3 digest of the data.
func Sum256(data []byte) (digest [32]byte) {
	hash := Sm3Sum(data)
	copy(digest[:], hash)
	return
}

// Sum calculate data into hash
func Sum(hash, data []byte) {
	tmp := Sm3Sum(data)
	copy(hash, tmp)
}
