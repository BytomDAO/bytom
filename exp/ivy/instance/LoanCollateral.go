package instance

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bytom/encoding/json"
	"github.com/bytom/exp/ivy/compiler"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/vm"
)

// LoanCollateralBodyBytes refer to contract's body
var LoanCollateralBodyBytes []byte

func init() {
	LoanCollateralBodyBytes, _ = hex.DecodeString("557a641c00000000007251567ac1695100c3c251567ac163280000007bc59f690000c3c251577ac1")
}

// contract LoanCollateral(assetLoaned: Asset, amountLoaned: Amount, repaymentDue: Time, lender: Program, borrower: Program) locks collateral
//
// 5                   [... <clause selector> borrower lender repaymentDue amountLoaned assetLoaned 5]
// ROLL                [... borrower lender repaymentDue amountLoaned assetLoaned <clause selector>]
// JUMPIF:$default     [... borrower lender repaymentDue amountLoaned assetLoaned]
// $repay              [... borrower lender repaymentDue amountLoaned assetLoaned]
// 0                   [... borrower lender repaymentDue amountLoaned assetLoaned 0]
// 0                   [... borrower lender repaymentDue amountLoaned assetLoaned 0 0]
// 3                   [... borrower lender repaymentDue amountLoaned assetLoaned 0 0 3]
// ROLL                [... borrower lender repaymentDue assetLoaned 0 0 amountLoaned]
// 3                   [... borrower lender repaymentDue assetLoaned 0 0 amountLoaned 3]
// ROLL                [... borrower lender repaymentDue 0 0 amountLoaned assetLoaned]
// 1                   [... borrower lender repaymentDue 0 0 amountLoaned assetLoaned 1]
// 6                   [... borrower lender repaymentDue 0 0 amountLoaned assetLoaned 1 6]
// ROLL                [... borrower repaymentDue 0 0 amountLoaned assetLoaned 1 lender]
// CHECKOUTPUT         [... borrower repaymentDue checkOutput(payment, lender)]
// VERIFY              [... borrower repaymentDue]
// 1                   [... borrower repaymentDue 1]
// 0                   [... borrower repaymentDue 1 0]
// AMOUNT              [... borrower repaymentDue 1 0 <amount>]
// ASSET               [... borrower repaymentDue 1 0 <amount> <asset>]
// 1                   [... borrower repaymentDue 1 0 <amount> <asset> 1]
// 6                   [... borrower repaymentDue 1 0 <amount> <asset> 1 6]
// ROLL                [... repaymentDue 1 0 <amount> <asset> 1 borrower]
// CHECKOUTPUT         [... repaymentDue checkOutput(collateral, borrower)]
// JUMP:$_end          [... borrower lender repaymentDue amountLoaned assetLoaned]
// $default            [... borrower lender repaymentDue amountLoaned assetLoaned]
// 2                   [... borrower lender repaymentDue amountLoaned assetLoaned 2]
// ROLL                [... borrower lender amountLoaned assetLoaned repaymentDue]
// BLOCKTIME LESSTHAN  [... borrower lender amountLoaned assetLoaned after(repaymentDue)]
// VERIFY              [... borrower lender amountLoaned assetLoaned]
// 0                   [... borrower lender amountLoaned assetLoaned 0]
// 0                   [... borrower lender amountLoaned assetLoaned 0 0]
// AMOUNT              [... borrower lender amountLoaned assetLoaned 0 0 <amount>]
// ASSET               [... borrower lender amountLoaned assetLoaned 0 0 <amount> <asset>]
// 1                   [... borrower lender amountLoaned assetLoaned 0 0 <amount> <asset> 1]
// 7                   [... borrower lender amountLoaned assetLoaned 0 0 <amount> <asset> 1 7]
// ROLL                [... borrower amountLoaned assetLoaned 0 0 <amount> <asset> 1 lender]
// CHECKOUTPUT         [... borrower amountLoaned assetLoaned checkOutput(collateral, lender)]
// $_end               [... borrower lender repaymentDue amountLoaned assetLoaned]

// PayToLoanCollateral instantiates contract LoanCollateral as a program with specific arguments.
func PayToLoanCollateral(assetLoaned bc.AssetID, amountLoaned uint64, repaymentDue time.Time, lender []byte, borrower []byte) ([]byte, error) {
	_contractParams := []*compiler.Param{
		{Name: "assetLoaned", Type: "Asset"},
		{Name: "amountLoaned", Type: "Amount"},
		{Name: "repaymentDue", Type: "Time"},
		{Name: "lender", Type: "Program"},
		{Name: "borrower", Type: "Program"},
	}
	var _contractArgs []compiler.ContractArg
	_assetLoaned := assetLoaned.Bytes()
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&_assetLoaned)})
	_amountLoaned := int64(amountLoaned)
	_contractArgs = append(_contractArgs, compiler.ContractArg{I: &_amountLoaned})
	_repaymentDue := repaymentDue.UnixNano() / int64(time.Millisecond)
	_contractArgs = append(_contractArgs, compiler.ContractArg{I: &_repaymentDue})
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&lender)})
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&borrower)})
	return compiler.Instantiate(LoanCollateralBodyBytes, _contractParams, false, _contractArgs)
}

// ParsePayToLoanCollateral parses the arguments out of an instantiation of contract LoanCollateral.
// If the input is not an instantiation of LoanCollateral, returns an error.
func ParsePayToLoanCollateral(prog []byte) ([][]byte, error) {
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
	if !bytes.Equal(LoanCollateralBodyBytes, insts[1].Data) {
		return nil, fmt.Errorf("body bytes do not match LoanCollateral")
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
