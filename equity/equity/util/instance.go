package equity

import (
	"encoding/hex"
	"errors"
	"strconv"

	chainjson "github.com/bytom/bytom/encoding/json"

	"github.com/bytom/bytom/equity/compiler"
)

// InstantiateContract instantiate contract parameters
func InstantiateContract(contract *compiler.Contract, args []compiler.ContractArg) ([]byte, error) {
	program, err := compiler.Instantiate(contract.Body, contract.Params, contract.Recursive, args)
	if err != nil {
		return nil, err
	}

	return program, nil
}

func ConvertArguments(contract *compiler.Contract, args []string) ([]compiler.ContractArg, error) {
	var contractArgs []compiler.ContractArg
	for i, p := range contract.Params {
		var argument compiler.ContractArg
		switch p.Type {
		case "Boolean":
			var boolValue bool
			if args[i] == "true" || args[i] == "1" {
				boolValue = true
			} else if args[i] == "false" || args[i] == "0" {
				boolValue = false
			} else {
				return nil, errors.New("mismatch Boolean argument")
			}
			argument.B = &boolValue

		case "Amount":
			amount, err := strconv.ParseUint(args[i], 10, 64)
			if err != nil {
				return nil, err
			}

			if amount > uint64(1<<uint(63)) {
				return nil, errors.New("the Amount argument exceeds max int64")
			}
			amountValue := int64(amount)
			argument.I = &amountValue

		case "Integer":
			integerValue, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return nil, err
			}
			argument.I = &integerValue

		case "Asset", "Hash", "PublicKey":
			if len(args[i]) != 64 {
				return nil, errors.New("mismatch length for Asset/Hash/PublicKey argument")
			}

			commonValue, err := hex.DecodeString(args[i])
			if err != nil {
				return nil, err
			}
			argument.S = (*chainjson.HexBytes)(&commonValue)

		case "Program":
			program, err := hex.DecodeString(args[i])
			if err != nil {
				return nil, err
			}
			argument.S = (*chainjson.HexBytes)(&program)

		case "String":
			strValue := []byte(args[i])
			argument.S = (*chainjson.HexBytes)(&strValue)

		}
		contractArgs = append(contractArgs, argument)
	}

	return contractArgs, nil
}
