package txbuilder

import (
	"math"
	"time"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc/types"
)

// NewBuilder return new TemplateBuilder instance
func NewBuilder(maxTime time.Time) *TemplateBuilder {
	return &TemplateBuilder{maxTime: maxTime}
}

// TemplateBuilder is struct of building transactions
type TemplateBuilder struct {
	base                *types.TxData
	inputs              []*types.TxInput
	outputs             []*types.TxOutput
	signingInstructions []*SigningInstruction
	minTime             time.Time
	maxTime             time.Time
	timeRange           uint64
	referenceData       []byte
	rollbacks           []func()
	callbacks           []func() error
}

// AddInput add inputs of transactions
func (b *TemplateBuilder) AddInput(in *types.TxInput, sigInstruction *SigningInstruction) error {
	if in.InputType() != types.CoinbaseInputType && in.Amount() > math.MaxInt64 {
		return errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", in.Amount())
	}
	b.inputs = append(b.inputs, in)
	b.signingInstructions = append(b.signingInstructions, sigInstruction)
	return nil
}

// AddOutput add outputs of transactions
func (b *TemplateBuilder) AddOutput(o *types.TxOutput) error {
	if o.Amount > math.MaxInt64 {
		return errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", o.Amount)
	}
	b.outputs = append(b.outputs, o)
	return nil
}

// InputCount return number of input in the template builder
func (b *TemplateBuilder) InputCount() int {
	return len(b.inputs)
}

// RestrictMinTime set minTime
func (b *TemplateBuilder) RestrictMinTime(t time.Time) {
	if t.After(b.minTime) {
		b.minTime = t
	}
}

// RestrictMaxTime set maxTime
func (b *TemplateBuilder) RestrictMaxTime(t time.Time) {
	if t.Before(b.maxTime) {
		b.maxTime = t
	}
}

// MaxTime return maxTime
func (b *TemplateBuilder) MaxTime() time.Time {
	return b.maxTime
}

// OnRollback registers a function that can be
// used to attempt to undo any side effects of building
// actions. For example, it might cancel any reservations
// reservations that were made on UTXOs in a spend action.
// Rollback is a "best-effort" operation and not guaranteed
// to succeed. Each action's side effects, if any, must be
// designed with this in mind.
func (b *TemplateBuilder) OnRollback(rollbackFn func()) {
	b.rollbacks = append(b.rollbacks, rollbackFn)
}

// OnBuild registers a function that will be run after all
// actions have been successfully built.
func (b *TemplateBuilder) OnBuild(buildFn func() error) {
	b.callbacks = append(b.callbacks, buildFn)
}

// Rollback action for handle fail build
func (b *TemplateBuilder) Rollback() {
	for _, f := range b.rollbacks {
		f()
	}
}

// Build build transactions with template
func (b *TemplateBuilder) Build() (*Template, *types.TxData, error) {
	// Run any building callbacks.
	for _, cb := range b.callbacks {
		err := cb()
		if err != nil {
			return nil, nil, err
		}
	}

	tpl := &Template{}
	tx := b.base
	if tx == nil {
		tx = &types.TxData{
			Version: 1,
		}
	}

	if b.timeRange != 0 {
		tx.TimeRange = b.timeRange
	}

	// Add all the built outputs.
	tx.Outputs = append(tx.Outputs, b.outputs...)

	// Add all the built inputs and their corresponding signing instructions.
	for i, in := range b.inputs {
		instruction := b.signingInstructions[i]
		instruction.Position = uint32(len(tx.Inputs))

		// Empty signature arrays should be serialized as empty arrays, not null.
		if instruction.WitnessComponents == nil {
			instruction.WitnessComponents = []witnessComponent{}
		}
		tpl.SigningInstructions = append(tpl.SigningInstructions, instruction)
		tx.Inputs = append(tx.Inputs, in)
	}

	tpl.Transaction = types.NewTx(*tx)
	return tpl, tx, nil
}
