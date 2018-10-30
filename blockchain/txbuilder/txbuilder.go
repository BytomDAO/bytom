// Package txbuilder builds a Chain Protocol transaction from
// a list of actions.
package txbuilder

import (
	"context"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/errors"
	"github.com/bytom/math/checked"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
)

// errors
var (
	//ErrBadRefData means invalid reference data
	ErrBadRefData = errors.New("transaction reference data does not match previous template's reference data")
	//ErrBadTxInputIdx means unsigned tx input
	ErrBadTxInputIdx = errors.New("unsigned tx missing input")
	//ErrBadWitnessComponent means invalid witness component
	ErrBadWitnessComponent = errors.New("invalid witness component")
	//ErrBadAmount means invalid asset amount
	ErrBadAmount = errors.New("bad asset amount")
	//ErrBlankCheck means unsafe transaction
	ErrBlankCheck = errors.New("unsafe transaction: leaves assets free to control")
	//ErrAction means errors occurred in actions
	ErrAction = errors.New("errors occurred in one or more actions")
	//ErrMissingFields means missing required fields
	ErrMissingFields = errors.New("required field is missing")
	//ErrBadContractArgType means invalid contract argument type
	ErrBadContractArgType = errors.New("invalid contract argument type")
)

// Build builds or adds on to a transaction.
// Initially, inputs are left unconsumed, and destinations unsatisfied.
// Build partners then satisfy and consume inputs and destinations.
// The final party must ensure that the transaction is
// balanced before calling finalize.
func Build(ctx context.Context, tx *types.TxData, actions []Action, maxTime time.Time, timeRange uint64) (*Template, error) {
	builder := TemplateBuilder{
		base:      tx,
		maxTime:   maxTime,
		timeRange: timeRange,
	}

	// Build all of the actions, updating the builder.
	var errs []error
	for i, action := range actions {
		err := action.Build(ctx, &builder)
		if err != nil {
			log.WithFields(log.Fields{"action index": i, "error": err}).Error("Loop tx's action")
			errs = append(errs, errors.WithDetailf(err, "action index %v", i))
		}
	}

	// If there were any errors, rollback and return a composite error.
	if len(errs) > 0 {
		builder.Rollback()
		return nil, errors.WithData(ErrAction, "actions", errs)
	}

	// Build the transaction template.
	tpl, tx, err := builder.Build()
	if err != nil {
		builder.Rollback()
		return nil, err
	}

	/*TODO: This part is use for check the balance, but now we are using btm as gas fee
	the rule need to be rewrite when we have time
	err = checkBlankCheck(tx)
	if err != nil {
		builder.rollback()
		return nil, err
	}*/

	return tpl, nil
}

// Sign will try to sign all the witness
func Sign(ctx context.Context, tpl *Template, auth string, signFn SignFunc) error {
	for i, sigInst := range tpl.SigningInstructions {
		for j, wc := range sigInst.WitnessComponents {
			switch sw := wc.(type) {
			case *SignatureWitness:
				err := sw.sign(ctx, tpl, uint32(i), auth, signFn)
				if err != nil {
					return errors.WithDetailf(err, "adding signature(s) to signature witness component %d of input %d", j, i)
				}
			case *RawTxSigWitness:
				err := sw.sign(ctx, tpl, uint32(i), auth, signFn)
				if err != nil {
					return errors.WithDetailf(err, "adding signature(s) to raw-signature witness component %d of input %d", j, i)
				}
			}
		}
	}
	return materializeWitnesses(tpl)
}

func checkBlankCheck(tx *types.TxData) error {
	assetMap := make(map[bc.AssetID]int64)
	var ok bool
	for _, in := range tx.Inputs {
		asset := in.AssetID() // AssetID() is calculated for IssuanceInputs, so grab once
		assetMap[asset], ok = checked.AddInt64(assetMap[asset], int64(in.Amount()))
		if !ok {
			return errors.WithDetailf(ErrBadAmount, "cumulative amounts for asset %x overflow the allowed asset amount 2^63", asset)
		}
	}
	for _, out := range tx.Outputs {
		assetMap[*out.AssetId], ok = checked.SubInt64(assetMap[*out.AssetId], int64(out.Amount))
		if !ok {
			return errors.WithDetailf(ErrBadAmount, "cumulative amounts for asset %x overflow the allowed asset amount 2^63", out.AssetId.Bytes())
		}
	}

	var requiresOutputs, requiresInputs bool
	for _, amt := range assetMap {
		if amt > 0 {
			requiresOutputs = true
		}
		if amt < 0 {
			requiresInputs = true
		}
	}

	// 4 possible cases here:
	// 1. requiresOutputs - false requiresInputs - false
	//    This is a balanced transaction with no free assets to consume.
	//    It could potentially be a complete transaction.
	// 2. requiresOutputs - true requiresInputs - false
	//    This is an unbalanced transaction with free assets to consume
	// 3. requiresOutputs - false requiresInputs - true
	//    This is an unbalanced transaction with a requiring assets to be spent
	// 4. requiresOutputs - true requiresInputs - true
	//    This is an unbalanced transaction with free assets to consume
	//    and requiring assets to be spent.
	// The only case that needs to be protected against is 2.
	if requiresOutputs && !requiresInputs {
		return errors.Wrap(ErrBlankCheck)
	}

	return nil
}

// MissingFieldsError returns a wrapped error ErrMissingFields
// with a data item containing the given field names.
func MissingFieldsError(name ...string) error {
	return errors.WithData(ErrMissingFields, "missing_fields", name)
}

// AddContractArgs add contract arguments
func AddContractArgs(sigInst *SigningInstruction, arguments []ContractArgument) error {
	for _, arg := range arguments {
		switch arg.Type {
		case "raw_tx_signature":
			rawTxSig := &RawTxSigArgument{}
			if err := json.Unmarshal(arg.RawData, rawTxSig); err != nil {
				return err
			}

			// convert path form chainjson.HexBytes to byte
			var path [][]byte
			for _, p := range rawTxSig.Path {
				path = append(path, []byte(p))
			}
			sigInst.AddRawWitnessKeys([]chainkd.XPub{rawTxSig.RootXPub}, path, 1)

		case "data":
			data := &DataArgument{}
			if err := json.Unmarshal(arg.RawData, data); err != nil {
				return err
			}
			sigInst.WitnessComponents = append(sigInst.WitnessComponents, DataWitness(data.Value))

		case "string":
			data := &StrArgument{}
			if err := json.Unmarshal(arg.RawData, data); err != nil {
				return err
			}
			sigInst.WitnessComponents = append(sigInst.WitnessComponents, DataWitness([]byte(data.Value)))

		case "integer":
			data := &IntegerArgument{}
			if err := json.Unmarshal(arg.RawData, data); err != nil {
				return err
			}
			sigInst.WitnessComponents = append(sigInst.WitnessComponents, DataWitness(vm.Int64Bytes(data.Value)))

		case "boolean":
			data := &BoolArgument{}
			if err := json.Unmarshal(arg.RawData, data); err != nil {
				return err
			}
			sigInst.WitnessComponents = append(sigInst.WitnessComponents, DataWitness(vm.BoolBytes(data.Value)))

		default:
			return ErrBadContractArgType
		}
	}

	return nil
}
