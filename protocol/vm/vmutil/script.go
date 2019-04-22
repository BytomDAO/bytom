package vmutil

import (
	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/vm"
)

// pre-define errors
var (
	ErrBadValue       = errors.New("bad value")
	ErrMultisigFormat = errors.New("bad multisig program format")
)

// IsUnspendable checks if a contorl program is absolute failed
func IsUnspendable(prog []byte) bool {
	return len(prog) > 0 && prog[0] == byte(vm.OP_FAIL)
}

func (b *Builder) addP2SPMultiSig(pubkeys []ed25519.PublicKey, nrequired int) error {
	if err := checkMultiSigParams(int64(nrequired), int64(len(pubkeys))); err != nil {
		return err
	}

	b.AddOp(vm.OP_TXSIGHASH) // stack is now [... NARGS SIG SIG SIG PREDICATEHASH]
	for _, p := range pubkeys {
		b.AddData(p)
	}
	b.AddInt64(int64(nrequired))    // stack is now [... SIG SIG SIG PREDICATEHASH PUB PUB PUB M]
	b.AddInt64(int64(len(pubkeys))) // stack is now [... SIG SIG SIG PREDICATEHASH PUB PUB PUB M N]
	b.AddOp(vm.OP_CHECKMULTISIG)    // stack is now [... NARGS]
	return nil
}

// DefaultCoinbaseProgram generates the script for contorl coinbase output
func DefaultCoinbaseProgram() ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_TRUE)
	return builder.Build()
}

// P2WPKHProgram return the segwit pay to public key hash
func P2WPKHProgram(hash []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddInt64(0)
	builder.AddData(hash)
	return builder.Build()
}

// P2WSHProgram return the segwit pay to script hash
func P2WSHProgram(hash []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddInt64(0)
	builder.AddData(hash)
	return builder.Build()
}

// RetireProgram generates the script for retire output
func RetireProgram(comment []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_FAIL)
	if len(comment) != 0 {
		builder.AddData(comment)
	}
	return builder.Build()
}

// P2PKHSigProgram generates the script for control with pubkey hash
func P2PKHSigProgram(pubkeyHash []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_HASH160)
	builder.AddData(pubkeyHash)
	builder.AddOp(vm.OP_EQUALVERIFY)
	builder.AddOp(vm.OP_TXSIGHASH)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_CHECKSIG)
	return builder.Build()
}

// P2SHProgram generates the script for control with script hash
func P2SHProgram(scriptHash []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_SHA3)
	builder.AddData(scriptHash)
	builder.AddOp(vm.OP_EQUALVERIFY)
	builder.AddInt64(-1)
	builder.AddOp(vm.OP_SWAP)
	builder.AddInt64(0)
	builder.AddOp(vm.OP_CHECKPREDICATE)
	return builder.Build()
}

// P2SPMultiSigProgram generates the script for control transaction output
func P2SPMultiSigProgram(pubkeys []ed25519.PublicKey, nrequired int) ([]byte, error) {
	builder := NewBuilder()
	if err := builder.addP2SPMultiSig(pubkeys, nrequired); err != nil {
		return nil, err
	}
	return builder.Build()
}

// P2SPMultiSigProgramWithHeight generates the script with block height for control transaction output
func P2SPMultiSigProgramWithHeight(pubkeys []ed25519.PublicKey, nrequired int, blockHeight int64) ([]byte, error) {
	builder := NewBuilder()
	if blockHeight > 0 {
		builder.AddInt64(blockHeight)
		builder.AddOp(vm.OP_BLOCKHEIGHT)
		builder.AddOp(vm.OP_GREATERTHAN)
		builder.AddOp(vm.OP_VERIFY)
	} else if blockHeight < 0 {
		return nil, errors.WithDetail(ErrBadValue, "negative blockHeight")
	}
	if err := builder.addP2SPMultiSig(pubkeys, nrequired); err != nil {
		return nil, err
	}
	return builder.Build()
}

func checkMultiSigParams(nrequired, npubkeys int64) error {
	if nrequired < 0 {
		return errors.WithDetail(ErrBadValue, "negative quorum")
	}
	if npubkeys < 0 {
		return errors.WithDetail(ErrBadValue, "negative pubkey count")
	}
	if nrequired > npubkeys {
		return errors.WithDetail(ErrBadValue, "quorum too big")
	}
	if nrequired == 0 && npubkeys > 0 {
		return errors.WithDetail(ErrBadValue, "quorum empty with non-empty pubkey list")
	}
	return nil
}

// GetIssuanceProgramRestrictHeight return issuance program restrict height
// if height invalid return 0
func GetIssuanceProgramRestrictHeight(program []byte) int64 {
	insts, err := vm.ParseProgram(program)
	if err != nil {
		return 0
	}

	if len(insts) >= 4 && insts[0].IsPushdata() && insts[1].Op == vm.OP_BLOCKHEIGHT && insts[2].Op == vm.OP_GREATERTHAN && insts[3].Op == vm.OP_VERIFY {
		height, err := vm.AsInt64(insts[0].Data)
		if err != nil {
			return 0
		}

		return height
	}
	return 0
}
