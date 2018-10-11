package txbuilder

import (
	"bytes"
	"context"

	cfg "github.com/bytom/config"
	"github.com/bytom/consensus"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
)

var (
	// ErrRejected means the network rejected a tx (as a double-spend)
	ErrRejected = errors.New("transaction rejected")
	// ErrMissingRawTx means missing transaction
	ErrMissingRawTx = errors.New("missing raw tx")
	// ErrBadInstructionCount means too many signing instructions compare with inputs
	ErrBadInstructionCount = errors.New("too many signing instructions in template")
	// ErrOrphanTx means submit transaction is orphan
	ErrOrphanTx = errors.New("finalize can't find transaction input utxo")
	// ErrExtTxFee means transaction fee exceed max limit
	ErrExtTxFee = errors.New("transaction fee exceed max limit")
)

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, c *protocol.Chain, tx *types.Tx) error {
	if fee := CalculateTxFee(tx); fee > cfg.CommonConfig.Wallet.MaxTxFee {
		return ErrExtTxFee
	}

	if err := checkTxSighashCommitment(tx); err != nil {
		return err
	}

	// This part is use for prevent tx size  is 0
	data, err := tx.TxData.MarshalText()
	if err != nil {
		return err
	}
	tx.TxData.SerializedSize = uint64(len(data))
	tx.Tx.SerializedSize = uint64(len(data))

	isOrphan, err := c.ValidateTx(tx)
	if errors.Root(err) == protocol.ErrBadTx {
		return errors.Sub(ErrRejected, err)
	}
	if err != nil {
		return errors.WithDetail(err, "tx rejected: "+err.Error())
	}
	if isOrphan {
		return ErrOrphanTx
	}
	return nil
}

var (
	// ErrNoTxSighashCommitment is returned when no input commits to the
	// complete transaction.
	// To permit idempotence of transaction submission, we require at
	// least one input to commit to the complete transaction (what you get
	// when you build a transaction with allow_additional_actions=false).
	ErrNoTxSighashCommitment = errors.New("no commitment to tx sighash")

	// ErrNoTxSighashAttempt is returned when there was no attempt made to sign
	// this transaction.
	ErrNoTxSighashAttempt = errors.New("no tx sighash attempted")

	// ErrTxSignatureFailure is returned when there was an attempt to sign this
	// transaction, but it failed.
	ErrTxSignatureFailure = errors.New("tx signature was attempted but failed")
)

func checkTxSighashCommitment(tx *types.Tx) error {
	// TODO: this is the local sender check rules, we might don't need it due to the rule is difference
	return nil
	var lastError error

	for i, inp := range tx.Inputs {
		var args [][]byte
		switch t := inp.TypedInput.(type) {
		case *types.SpendInput:
			args = t.Arguments
		case *types.IssuanceInput:
			args = t.Arguments
		}
		// Note: These numbers will need to change if more args are added such that the minimum length changes
		switch {
		// A conforming arguments list contains
		// [... arg1 arg2 ... argN N sig1 sig2 ... sigM prog]
		// The args are the opaque arguments to prog. In the case where
		// N is 0 (prog takes no args), and assuming there must be at
		// least one signature, args has a minimum length of 3.
		case len(args) == 0:
			lastError = ErrNoTxSighashAttempt
			continue
		case len(args) < 3:
			lastError = ErrTxSignatureFailure
			continue
		}
		lastError = ErrNoTxSighashCommitment
		prog := args[len(args)-1]
		if len(prog) != 35 {
			continue
		}
		if prog[0] != byte(vm.OP_DATA_32) {
			continue
		}
		if !bytes.Equal(prog[33:], []byte{byte(vm.OP_TXSIGHASH), byte(vm.OP_EQUAL)}) {
			continue
		}
		h := tx.SigHash(uint32(i))
		if !bytes.Equal(h.Bytes(), prog[1:33]) {
			continue
		}
		// At least one input passes commitment checks
		return nil
	}

	return lastError
}

// CalculateTxFee calculate transaction fee
func CalculateTxFee(tx *types.Tx) (fee uint64) {
	totalInputBTM := uint64(0)
	totalOutputBTM := uint64(0)

	for _, input := range tx.Inputs {
		if input.InputType() != types.CoinbaseInputType && input.AssetID() == *consensus.BTMAssetID {
			totalInputBTM += input.Amount()
		}
	}

	for _, output := range tx.Outputs {
		if *output.AssetId == *consensus.BTMAssetID {
			totalOutputBTM += output.Amount
		}
	}

	fee = totalInputBTM - totalOutputBTM
	return
}
