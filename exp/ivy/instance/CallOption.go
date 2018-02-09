package instance

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/encoding/json"
	"github.com/bytom/exp/ivy/compiler"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/vm"
)

// CallOptionBodyBytes refer to contract's body
var CallOptionBodyBytes []byte

func init() {
	CallOptionBodyBytes, _ = hex.DecodeString("557a6422000000547ac5a069547a547aae7cac6900007b537a51557ac1632f000000547ac59f690000c3c251577ac1")
}

// contract CallOption(strikePrice: Amount, strikeCurrency: Asset, seller: Program, buyerKey: PublicKey, deadline: Time) locks underlying
//
// 5                        [... <clause selector> deadline buyerKey seller strikeCurrency strikePrice 5]
// ROLL                     [... deadline buyerKey seller strikeCurrency strikePrice <clause selector>]
// JUMPIF:$expire           [... deadline buyerKey seller strikeCurrency strikePrice]
// $exercise                [... deadline buyerKey seller strikeCurrency strikePrice]
// 4                        [... buyerSig deadline buyerKey seller strikeCurrency strikePrice 4]
// ROLL                     [... buyerSig buyerKey seller strikeCurrency strikePrice deadline]
// BLOCKTIME GREATERTHAN    [... buyerSig buyerKey seller strikeCurrency strikePrice before(deadline)]
// VERIFY                   [... buyerSig buyerKey seller strikeCurrency strikePrice]
// 4                        [... buyerSig buyerKey seller strikeCurrency strikePrice 4]
// ROLL                     [... buyerKey seller strikeCurrency strikePrice buyerSig]
// 4                        [... buyerKey seller strikeCurrency strikePrice buyerSig 4]
// ROLL                     [... seller strikeCurrency strikePrice buyerSig buyerKey]
// TXSIGHASH SWAP CHECKSIG  [... seller strikeCurrency strikePrice checkTxSig(buyerKey, buyerSig)]
// VERIFY                   [... seller strikeCurrency strikePrice]
// 0                        [... seller strikeCurrency strikePrice 0]
// 0                        [... seller strikeCurrency strikePrice 0 0]
// 2                        [... seller strikeCurrency strikePrice 0 0 2]
// ROLL                     [... seller strikeCurrency 0 0 strikePrice]
// 3                        [... seller strikeCurrency 0 0 strikePrice 3]
// ROLL                     [... seller 0 0 strikePrice strikeCurrency]
// 1                        [... seller 0 0 strikePrice strikeCurrency 1]
// 5                        [... seller 0 0 strikePrice strikeCurrency 1 5]
// ROLL                     [... 0 0 strikePrice strikeCurrency 1 seller]
// CHECKOUTPUT              [... checkOutput(payment, seller)]
// JUMP:$_end               [... deadline buyerKey seller strikeCurrency strikePrice]
// $expire                  [... deadline buyerKey seller strikeCurrency strikePrice]
// 4                        [... deadline buyerKey seller strikeCurrency strikePrice 4]
// ROLL                     [... buyerKey seller strikeCurrency strikePrice deadline]
// BLOCKTIME LESSTHAN       [... buyerKey seller strikeCurrency strikePrice after(deadline)]
// VERIFY                   [... buyerKey seller strikeCurrency strikePrice]
// 0                        [... buyerKey seller strikeCurrency strikePrice 0]
// 0                        [... buyerKey seller strikeCurrency strikePrice 0 0]
// AMOUNT                   [... buyerKey seller strikeCurrency strikePrice 0 0 <amount>]
// ASSET                    [... buyerKey seller strikeCurrency strikePrice 0 0 <amount> <asset>]
// 1                        [... buyerKey seller strikeCurrency strikePrice 0 0 <amount> <asset> 1]
// 7                        [... buyerKey seller strikeCurrency strikePrice 0 0 <amount> <asset> 1 7]
// ROLL                     [... buyerKey strikeCurrency strikePrice 0 0 <amount> <asset> 1 seller]
// CHECKOUTPUT              [... buyerKey strikeCurrency strikePrice checkOutput(underlying, seller)]
// $_end                    [... deadline buyerKey seller strikeCurrency strikePrice]

// PayToCallOption instantiates contract CallOption as a program with specific arguments.
func PayToCallOption(strikePrice uint64, strikeCurrency bc.AssetID, seller []byte, buyerKey ed25519.PublicKey, deadline time.Time) ([]byte, error) {
	_contractParams := []*compiler.Param{
		{Name: "strikePrice", Type: "Amount"},
		{Name: "strikeCurrency", Type: "Asset"},
		{Name: "seller", Type: "Program"},
		{Name: "buyerKey", Type: "PublicKey"},
		{Name: "deadline", Type: "Time"},
	}
	var _contractArgs []compiler.ContractArg
	_strikePrice := int64(strikePrice)
	_contractArgs = append(_contractArgs, compiler.ContractArg{I: &_strikePrice})
	_strikeCurrency := strikeCurrency.Bytes()
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&_strikeCurrency)})
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&seller)})
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&buyerKey)})
	_deadline := deadline.UnixNano() / int64(time.Millisecond)
	_contractArgs = append(_contractArgs, compiler.ContractArg{I: &_deadline})
	return compiler.Instantiate(CallOptionBodyBytes, _contractParams, false, _contractArgs)
}

// ParsePayToCallOption parses the arguments out of an instantiation of contract CallOption.
// If the input is not an instantiation of CallOption, returns an error.
func ParsePayToCallOption(prog []byte) ([][]byte, error) {
	var result [][]byte
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 5; i++ {
		if len(insts) == 0 {
			return nil, fmt.Errorf("program too short")
		}
		if !insts[0].IsPushdata() {
			return nil, fmt.Errorf("too few arguments")
		}
		result = append(result, insts[0].Data)
		insts = insts[1:]
	}
	if len(insts) != 4 {
		return nil, fmt.Errorf("program too short")
	}
	if insts[0].Op != vm.OP_DEPTH {
		return nil, fmt.Errorf("wrong program format")
	}
	if !insts[1].IsPushdata() {
		return nil, fmt.Errorf("wrong program format")
	}
	if !bytes.Equal(CallOptionBodyBytes, insts[1].Data) {
		return nil, fmt.Errorf("body bytes do not match CallOption")
	}
	if !insts[2].IsPushdata() {
		return nil, fmt.Errorf("wrong program format")
	}
	v, err := vm.AsInt64(insts[2].Data)
	if err != nil {
		return nil, err
	}
	if v != 0 {
		return nil, fmt.Errorf("wrong program format")
	}
	if insts[3].Op != vm.OP_CHECKPREDICATE {
		return nil, fmt.Errorf("wrong program format")
	}
	return result, nil
}
