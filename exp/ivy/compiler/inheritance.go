package compiler

import (
	"errors"
	"fmt"
	"strings"
)

func parseInheritance(p *parser) []string {
	var inheritance []string
	consumeKeyword(p, "extends")
	first := true
	for !peekTok(p, "{") {
		if first {
			first = false
		} else {
			consumeTok(p, ",")
		}
		contractName := consumeIdentifier(p)
		inheritance = append(inheritance, contractName)
	}

	return inheritance
}

func addInheritClause(contract *Contract, contracts []*Contract) error {
	if len(contract.Inheritance) != 0 {
		var result []*Clause

		// Add the inheritance contract into current contract
		for _, inherit := range contract.Inheritance {
			inherit = strings.TrimSpace(inherit)

			finded := false
			for _, c := range contracts {
				if c.Name == inherit {
					finded = true

					if c.Value != contract.Value {
						errMsg := fmt.Sprintf("the locks value for inherit contract [%s:%s] is not equal to the current contract [%s:%s] !",
							c.Name, c.Value, contract.Name, contract.Value)
						return errors.New(errMsg)
					}

					for _, baseClause := range c.Clauses {
						result = append(result, baseClause)
					}
				}
			}

			if finded == false {
				errMsg := fmt.Sprintf("Not find the contract [%s]!", contract.Inheritance)
				return errors.New(errMsg)
			}
		}

		// After add the inherited contract, The order of clause is: base_clause ... current_clause
		for _, currentClause := range contract.Clauses {
			result = append(result, currentClause)
		}

		contract.Clauses = result[:]
	}

	return nil
}
