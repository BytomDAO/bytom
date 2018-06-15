package types

type RetireInput struct {
	Arbitrary string
}

// NewRetireInput create a new RetireInput struct.
func NewRetireInput(arbitrary string) *TxInput {
	return &TxInput{
		AssetVersion: 1,
		TypedInput:   &RetireInput{Arbitrary: arbitrary},
	}
}

// RetireInput is the interface function for return the input type.
func (si *RetireInput) InputType() uint8 { return RetireInputType }
