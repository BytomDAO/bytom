package config

/*func TestGenerateGenesisBlock(t *testing.T) {
	block := GenerateGenesisBlock()
	nonce := block.Nonce
	for {
		hash := block.Hash()
		if difficulty.CheckProofOfWork(&hash, consensus.InitialSeed, block.Bits) {
			break
		}
		block.Nonce++
	}
	if block.Nonce != nonce {
		t.Errorf("correct nonce is %d, but get %d", block.Nonce, nonce)
	}
}*/
