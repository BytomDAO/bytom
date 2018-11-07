package equity

import (
	"fmt"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/vm"

	"github.com/bytom/equity/compiler"
)

func Estimate(contract *compiler.Contract) (map[string]int64, error) {
	contractParamGas, err := calContractParamGas(contract.Params)
	if err != nil {
		return nil, err
	}

	clauseParamGasMap, err := calClauseParamGas(contract.Clauses)
	if err != nil {
		return nil, err
	}

	clauseGasMap, err := calClauseGas(contract, contractParamGas, clauseParamGasMap)
	if err != nil {
		return nil, err
	}

	return clauseGasMap, nil
}

func calContractParamGas(params []*compiler.Param) (int64, error) {
	contractParamGas := int64(0)
	for _, param := range params {
		if gas := vm.GetContractParamGas(string(param.Type)); gas != -1 {
			contractParamGas = contractParamGas + gas
		} else {
			err := errors.New("Invalid contract parameter type")
			return 0, err
		}
	}

	return contractParamGas, nil
}

func calClauseParamGas(clauses []*compiler.Clause) (map[string]int64, error) {
	clauseParamGasMap := make(map[string]int64)
	for _, clause := range clauses {
		clauseParamGas := int64(0)
		for _, param := range clause.Params {
			if fgas := vm.GetClauseParamGas(string(param.Type)); fgas != -1 {
				clauseParamGas = clauseParamGas + fgas
			} else {
				err := errors.New("Invalid contract's clause parameter type")
				return nil, err
			}
		}
		clauseParamGasMap[clause.Name] = clauseParamGas
	}

	return clauseParamGasMap, nil
}

func calClauseGas(contract *compiler.Contract, contractParamGas int64, clauseParamGasMap map[string]int64) (map[string]int64, error) {
	result, err := calculate(contract.Body)
	if err != nil {
		return nil, err
	}

	if len(result) != len(contract.Clauses) {
		errmsg := fmt.Sprintf("the length of result[%d] is not equal to the number of clause[%d]\n", len(result), len(contract.Clauses))
		err := errors.New(errmsg)
		return nil, err
	}

	var strParamList string
	clauseGasMap := make(map[string]int64)
	for i, clause := range contract.Clauses {
		for j, param := range clause.Params {
			if j != len(clause.Params)-1 {
				strParamList = strParamList + string(param.Type) + ", "
			} else {
				strParamList = strParamList + string(param.Type)
			}
		}

		clauseListName := fmt.Sprintf("%s(%s)", clause.Name, strParamList)
		clauseGasMap[clauseListName] = contractParamGas + clauseParamGasMap[clause.Name] + result[i]
	}
	return clauseGasMap, nil
}

func calculate(prog []byte) ([]int64, error) {
	//init the gas of instruction
	vm.InitGas()

	instructions, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	var clauseResult []int64
	var childClauseResult []int64
	result := int64(0)
	instGas := int64(0)
	count := 0
	intermediate := int64(0)

	//calculate the instruction consumed gas
	for i, inst := range instructions {
		switch inst.Op.String() {
		case "PUSHDATA1":
			if len(inst.Data) != 0 {
				instGas = int64(10 + len(inst.Data))
			} else {
				instGas = vm.GetGas(inst.Op)
			}
		case "PUSHDATA2":
			if len(inst.Data) != 0 {
				instGas = int64(11 + len(inst.Data))
			} else {
				instGas = vm.GetGas(inst.Op)
			}
		case "PUSHDATA4":
			if len(inst.Data) != 0 {
				instGas = int64(13 + len(inst.Data))
			} else {
				instGas = vm.GetGas(inst.Op)
			}
		case "CHECKPREDICATE":
			childprog := instructions[i-2].Data
			fmt.Println("\nstart childVM instructions")
			tmpclauseResult, err := calculate(childprog)
			if err != nil {
				fmt.Println("ParseProgram in childVM err:", err)
				return nil, err
			}
			for _, tmp := range tmpclauseResult {
				childClauseResult = append(childClauseResult, tmp)
			}
			fmt.Println("end childVM instructions")
			fmt.Printf("The result of childVM estimate gas: %v\n\n", childClauseResult)
			instGas = vm.GetGas(inst.Op)
		case "JUMPIF":
			instGas = vm.GetGas(inst.Op)
			//fmt.Printf("%v:  %d\n", inst.Op.String(), gas)
			if instructions[i+1].Op.String() != "JUMPIF" {
				intermediate = result + instGas
				result = 0
				instGas = 0
				//fmt.Printf("intermediate result: %d\n", intermediate)
			}
		case "JUMP":
			count = count + 1
			instGas = vm.GetGas(inst.Op)
			//fmt.Printf("%v:  %d\n", inst.Op.String(), gas)
			result = intermediate + result + instGas
			//fmt.Printf("the %d clause estimate gas: %d\n", count, result)
			clauseResult = append(clauseResult, result)
			result = 0
			instGas = 0
		default:
			instGas = vm.GetGas(inst.Op)
		}

		//if inst.Op.String() != "JUMP" && inst.Op.String() != "JUMPIF" {
		//	fmt.Printf("%v:  %d\n", inst.Op.String(), gas)
		//}
		result = result + instGas
	}

	if len(childClauseResult) > 0 {
		for i, _ := range childClauseResult {
			childClauseResult[i] = childClauseResult[i] + result
		}
		clauseResult = childClauseResult
	} else {
		//fmt.Println("The ending clause(or only one clause) estimate gas:", result)
		clauseResult = append(clauseResult, result)
	}

	return clauseResult, nil
}

func estimate(contract *compiler.Contract) error {
	//claculate the contract paraments consumed gas
	var contractParamGas int64
	contractParamGas = 0
	for _, cparam := range contract.Params {
		if cgas := vm.GetContractParamGas(string(cparam.Type)); cgas != -1 {
			contractParamGas = contractParamGas + cgas
		} else {
			errmsg := fmt.Sprintf("the type of contract parament [%v] is error\n", cparam.Type)
			err := errors.New(errmsg)
			return err
		}
	}
	fmt.Println("contractParamGas:", contractParamGas)

	//claculate the clause paraments consumed gas
	var clauseParamGasList []int64
	for i, _ := range contract.Clauses {
		clauseParamGas := int64(0)
		for _, fparam := range contract.Clauses[i].Params {
			if fgas := vm.GetClauseParamGas(string(fparam.Type)); fgas != -1 {
				clauseParamGas = clauseParamGas + fgas
			} else {
				errmsg := fmt.Sprintf("the type of clause parament [%v] is error\n", fparam.Type)
				err := errors.New(errmsg)
				return err
			}
		}
		clauseParamGasList = append(clauseParamGasList, clauseParamGas)
	}

	//print the clause paraments consumed gas
	fmt.Println("clauseParamGas:")
	for i, _ := range clauseParamGasList {
		clause := fmt.Sprintf("%s", contract.Clauses[i].Name)
		fmt.Printf("    %v:  %v\n", clause, clauseParamGasList[i])
	}

	//estimate gas
	result, err := calculate(contract.Body)
	if err != nil {
		return err
	}

	if len(result) != len(clauseParamGasList) {
		errmsg := fmt.Sprintf("the length of result[%d] is not equal to the number of clause[%d]\n", len(result), len(clauseParamGasList))
		err := errors.New(errmsg)
		return err
	}

	//print the estimation result
	fmt.Println("\nEstimation result:")
	for i, _ := range result {
		//print the clause paraments type
		var paramlist string
		for j, p := range contract.Clauses[i].Params {
			if j != len(contract.Clauses[i].Params)-1 {
				paramlist = paramlist + string(p.Type) + ", "
			} else {
				paramlist = paramlist + string(p.Type)
			}

		}

		clause := fmt.Sprintf("%s(%s)", contract.Clauses[i].Name, paramlist)
		fmt.Printf("    %v:  %v\n", clause, result[i]+contractParamGas+clauseParamGasList[i])
	}

	fmt.Println("\nNOTICE: \n    Estimated results for reference only, Please check the execution program consumed gas!!!")
	return nil
}
