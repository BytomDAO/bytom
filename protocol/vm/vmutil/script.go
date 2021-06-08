package vmutil

import (
	"crypto/ed25519"

	"github.com/bytom/bytom/consensus/bcrp"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/vm"
)

// pre-define errors
var (
	ErrBadValue       = errors.New("bad value")
	ErrMultisigFormat = errors.New("bad multisig program format")
)

// swapContractArgs is a struct for swap contract arguments
type SwapContractArgs struct {
	RequestedAsset0 bc.AssetID
	RequestedAsset1 bc.AssetID
	RequestedAsset2 bc.AssetID
}

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

// RegisterProgram generates the script for register contract output
// follow BCRP(bytom contract register protocol)
func RegisterProgram(contract []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_FAIL)
	builder.AddOp(vm.OP_PUSHDATA1)
	builder.AddData([]byte(bcrp.BCRP))
	builder.AddOp(vm.OP_PUSHDATA1)
	builder.AddData([]byte{byte(bcrp.Version)})
	builder.AddOp(vm.OP_PUSHDATA1)
	builder.AddData(contract)
	return builder.Build()
}

// CallContractProgram generates the script for control contract output
// follow BCRP(bytom contract register protocol)
func CallContractProgram(hash []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_PUSHDATA1)
	builder.AddData(hash)
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
func GetIssuanceProgramRestrictHeight(program []byte) uint64 {
	insts, err := vm.ParseProgram(program)
	if err != nil {
		return 0
	}

	if len(insts) >= 4 && insts[0].IsPushdata() && insts[1].Op == vm.OP_BLOCKHEIGHT && insts[2].Op == vm.OP_GREATERTHAN && insts[3].Op == vm.OP_VERIFY {
		heightInt, err := vm.AsBigInt(insts[0].Data)
		if err != nil {
			return 0
		}

		height, overflow := heightInt.Uint64WithOverflow()
		if overflow {
			return 0
		}

		return height
	}
	return 0
}

// P2WSCProgram return the segwit pay to swap contract
func P2WSCProgram(swapContractArgs SwapContractArgs) ([]byte, error) {
	builder := NewBuilder()
	builder.AddInt64(0)
	builder.AddData(swapContractArgs.RequestedAsset0.Bytes())
	builder.AddData(swapContractArgs.RequestedAsset1.Bytes())
	builder.AddData(swapContractArgs.RequestedAsset2.Bytes())
	return builder.Build()
}

// P2SCProgram generates the script for control with swap contract
//
// swapContract source code:
//
// contract stack flow:
func P2SCProgram(swapContractArgs SwapContractArgs) ([]byte, error) {
	program, err := P2WSCProgram(swapContractArgs)
	if err != nil {
		return nil, err
	}

	builder := NewBuilder()
	// contract arguments
	builder.AddData(program)

	// TODO: contract instructions
	return builder.Build()
}

func P2SCProgram0() ([]byte, error) {

	builder := NewBuilder()
	// contract arguments

	// contract instructions
	// altstack [1027 809698 00e8764817]
	builder.AddOp(vm.OP_FROMALTSTACK)
	builder.AddOp(vm.OP_FROMALTSTACK)
	builder.AddOp(vm.OP_FROMALTSTACK) //[00e8764817 809698 1027]
	builder.AddInt64(1000)            //[00e8764817 809698 1027 e803]
	builder.AddOp(vm.OP_ADD)          // [00e8764817 809698 1027 f82a]
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_TOALTSTACK)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_PICK) // [00e8764817 809698 f82a 00e8764817]
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_DIV) // [00e8764817 809698 efcd]  newPY
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_TOALTSTACK)
	//builder.AddOp(vm.OP_DROP)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_DROP)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_TOALTSTACK)
	// uint64(index), amount, assetID, uint64(vmVersion), code, vm.altStack
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	//builder.AddOp(vm.OP_2)
	//builder.AddOp(vm.OP_ASSET) // asset 2
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_4)
	builder.AddOp(vm.OP_ROLL)
	//builder.AddOp(vm.OP_CATPUSHDATA)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	//builder.AddOp(vm.OP_VERIFY)

	return builder.Build()
}

func P2SCProgram1() ([]byte, error) {

	builder := NewBuilder()
	// contract arguments

	// contract instructions
	//builder.AddOp(vm.OP_7)
	//builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_TOALTSTACK)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_NUMEQUAL)
	builder.AddJumpIf(0)
	builder.AddJumpIf(1)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	//builder.AddOp(vm.OP_MULFRACTION)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_OVER)
	builder.AddOp(vm.OP_0)
	builder.AddOp(vm.OP_GREATERTHANOREQUAL)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_LESSTHAN)
	builder.AddOp(vm.OP_BOOLAND)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_FROMALTSTACK)
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_TOALTSTACK)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddInt64(999)
	builder.AddInt64(1000)
	//builder.AddOp(vm.OP_MULFRACTION)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_5)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_FROMALTSTACK)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_ADD)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_SUB)
	builder.AddOp(vm.OP_ASSET)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_4)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddJump(2)
	builder.SetJumpTarget(1)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	//builder.AddOp(vm.OP_MULFRACTION)
	builder.AddInt64(999)
	builder.AddInt64(1000)
	//builder.AddOp(vm.OP_MULFRACTION)
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_0)
	builder.AddOp(vm.OP_GREATERTHANOREQUAL)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_FROMALTSTACK)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_4)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddJump(3)
	builder.SetJumpTarget(0)
	builder.AddOp(vm.OP_DROP)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_TXSIGHASH)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_CHECKSIG)
	builder.SetJumpTarget(2)
	builder.SetJumpTarget(3)
	return builder.Build()
}
