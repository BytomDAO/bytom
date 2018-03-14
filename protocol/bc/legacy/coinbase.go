package legacy

// CoinbaseInput is record the coinbase message
type CoinbaseInput struct {
	Arbitrary []byte
}

// IsIssuance is the interface function for return the struct type
func (cb *CoinbaseInput) IsIssuance() bool { return false }

// IsCoinbase is the interface function for return the struct type
func (cb *CoinbaseInput) IsCoinbase() bool { return true }

// NewCoinbaseInput create a new coinbase input struct
func NewCoinbaseInput(arbitrary []byte) *TxInput {
	return &TxInput{
		AssetVersion:  1,
		TypedInput: &CoinbaseInput{
			Arbitrary: arbitrary,
		},
	}
}
