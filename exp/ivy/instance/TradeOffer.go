package instance

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/encoding/json"
	"github.com/bytom/exp/ivy/compiler"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/vm"
)

// TradeOfferBodyBytes refer to contract's body
var TradeOfferBodyBytes []byte

func init() {
	TradeOfferBodyBytes, _ = hex.DecodeString("547a641300000000007251557ac1631a000000547a547aae7cac")
}

// contract TradeOffer(assetRequested: Asset, amountRequested: Amount, seller: Program, cancelKey: PublicKey) locks offered
//
// 4                        [... <clause selector> cancelKey seller amountRequested assetRequested 4]
// ROLL                     [... cancelKey seller amountRequested assetRequested <clause selector>]
// JUMPIF:$cancel           [... cancelKey seller amountRequested assetRequested]
// $trade                   [... cancelKey seller amountRequested assetRequested]
// 0                        [... cancelKey seller amountRequested assetRequested 0]
// 0                        [... cancelKey seller amountRequested assetRequested 0 0]
// 3                        [... cancelKey seller amountRequested assetRequested 0 0 3]
// ROLL                     [... cancelKey seller assetRequested 0 0 amountRequested]
// 3                        [... cancelKey seller assetRequested 0 0 amountRequested 3]
// ROLL                     [... cancelKey seller 0 0 amountRequested assetRequested]
// 1                        [... cancelKey seller 0 0 amountRequested assetRequested 1]
// 5                        [... cancelKey seller 0 0 amountRequested assetRequested 1 5]
// ROLL                     [... cancelKey 0 0 amountRequested assetRequested 1 seller]
// CHECKOUTPUT              [... cancelKey checkOutput(payment, seller)]
// JUMP:$_end               [... cancelKey seller amountRequested assetRequested]
// $cancel                  [... cancelKey seller amountRequested assetRequested]
// 4                        [... sellerSig cancelKey seller amountRequested assetRequested 4]
// ROLL                     [... cancelKey seller amountRequested assetRequested sellerSig]
// 4                        [... cancelKey seller amountRequested assetRequested sellerSig 4]
// ROLL                     [... seller amountRequested assetRequested sellerSig cancelKey]
// TXSIGHASH SWAP CHECKSIG  [... seller amountRequested assetRequested checkTxSig(cancelKey, sellerSig)]
// $_end                    [... cancelKey seller amountRequested assetRequested]

// PayToTradeOffer instantiates contract TradeOffer as a program with specific arguments.
func PayToTradeOffer(assetRequested bc.AssetID, amountRequested uint64, seller []byte, cancelKey ed25519.PublicKey) ([]byte, error) {
	_contractParams := []*compiler.Param{
		{Name: "assetRequested", Type: "Asset"},
		{Name: "amountRequested", Type: "Amount"},
		{Name: "seller", Type: "Program"},
		{Name: "cancelKey", Type: "PublicKey"},
	}
	var _contractArgs []compiler.ContractArg
	_assetRequested := assetRequested.Bytes()
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&_assetRequested)})
	_amountRequested := int64(amountRequested)
	_contractArgs = append(_contractArgs, compiler.ContractArg{I: &_amountRequested})
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&seller)})
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&cancelKey)})
	return compiler.Instantiate(TradeOfferBodyBytes, _contractParams, false, _contractArgs)
}

// ParsePayToTradeOffer parses the arguments out of an instantiation of contract TradeOffer.
// If the input is not an instantiation of TradeOffer, returns an error.
func ParsePayToTradeOffer(prog []byte) ([][]byte, error) {
	var result [][]byte
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 4; i++ {
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
	if !bytes.Equal(TradeOfferBodyBytes, insts[1].Data) {
		return nil, fmt.Errorf("body bytes do not match TradeOffer")
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
