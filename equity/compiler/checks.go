package compiler

import "fmt"

func checkRecursive(contract *Contract) bool {
	for _, clause := range contract.Clauses {
		for _, stmt := range clause.statements {
			if result := checkStatRecursive(stmt, contract.Name); result {
				return true
			}
		}
	}
	return false
}

func checkStatRecursive(stmt statement, contractName string) bool {
	switch s := stmt.(type) {
	case *ifStatement:
		for _, trueStmt := range s.body.trueBody {
			if result := checkStatRecursive(trueStmt, contractName); result {
				return true
			}
		}

		for _, falseStmt := range s.body.falseBody {
			if result := checkStatRecursive(falseStmt, contractName); result {
				return true
			}
		}

	case *lockStatement:
		if c, ok := s.program.(*callExpr); ok {
			if references(c.fn, contractName) {
				return true
			}
		}
	}

	return false
}

func prohibitSigParams(contract *Contract) error {
	for _, p := range contract.Params {
		if p.Type == sigType {
			return fmt.Errorf("contract parameter \"%s\" has type Signature, but contract parameters cannot have type Signature", p.Name)
		}
	}
	return nil
}

func requireAllParamsUsedInClauses(params []*Param, clauses []*Clause) error {
	for _, p := range params {
		used := false
		for _, c := range clauses {
			err := requireAllParamsUsedInClause([]*Param{p}, c)
			if err == nil {
				used = true
				break
			}
		}

		if !used {
			return fmt.Errorf("parameter \"%s\" is unused", p.Name)
		}
	}
	return nil
}

func requireAllParamsUsedInClause(params []*Param, clause *Clause) error {
	for _, p := range params {
		used := false
		for _, stmt := range clause.statements {
			if used = checkParamUsedInStatement(p, stmt); used {
				break
			}
		}

		if !used {
			return fmt.Errorf("parameter \"%s\" is unused in clause \"%s\"", p.Name, clause.Name)
		}
	}
	return nil
}

func checkParamUsedInStatement(param *Param, stmt statement) (used bool) {
	switch s := stmt.(type) {
	case *ifStatement:
		if used = references(s.condition, param.Name); used {
			return used
		}

		for _, st := range s.body.trueBody {
			if used = checkParamUsedInStatement(param, st); used {
				break
			}
		}

		if !used {
			for _, st := range s.body.falseBody {
				if used = checkParamUsedInStatement(param, st); used {
					break
				}
			}
		}

	case *defineStatement:
		used = references(s.expr, param.Name)
	case *assignStatement:
		used = references(s.expr, param.Name)
	case *verifyStatement:
		used = references(s.expr, param.Name)
	case *lockStatement:
		used = references(s.lockedAmount, param.Name) || references(s.lockedAsset, param.Name) || references(s.program, param.Name)
	case *unlockStatement:
		used = references(s.unlockedAmount, param.Name) || references(s.unlockedAsset, param.Name)
	}

	return used
}

func references(expr expression, name string) bool {
	switch e := expr.(type) {
	case *binaryExpr:
		return references(e.left, name) || references(e.right, name)
	case *unaryExpr:
		return references(e.expr, name)
	case *callExpr:
		if references(e.fn, name) {
			return true
		}
		for _, a := range e.args {
			if references(a, name) {
				return true
			}
		}
		return false
	case varRef:
		return string(e) == name
	case listExpr:
		for _, elt := range []expression(e) {
			if references(elt, name) {
				return true
			}
		}
		return false
	}
	return false
}

func referencedBuiltin(expr expression) *builtin {
	if v, ok := expr.(varRef); ok {
		for _, b := range builtins {
			if string(v) == b.name {
				return &b
			}
		}
	}
	return nil
}

func assignIndexes(clause *Clause) error {
	var nextIndex int64
	for i, stmt := range clause.statements {
		if nextIndex = assignStatIndexes(stmt, nextIndex, i != len(clause.statements)-1); nextIndex < 0 {
			return fmt.Errorf("Not support that the number of lock/unlock statement is not equal between ifbody and elsebody when the if-else is not the last statement in clause \"%s\"", clause.Name)
		}
	}

	return nil
}

func assignStatIndexes(stat statement, nextIndex int64, nonFinalFlag bool) int64 {
	switch stmt := stat.(type) {
	case *ifStatement:
		trueIndex := nextIndex
		falseIndex := nextIndex
		for _, trueStmt := range stmt.body.trueBody {
			trueIndex = assignStatIndexes(trueStmt, trueIndex, nonFinalFlag)
		}

		for _, falseStmt := range stmt.body.falseBody {
			falseIndex = assignStatIndexes(falseStmt, falseIndex, nonFinalFlag)
		}

		if trueIndex != falseIndex && nonFinalFlag {
			return -1
		} else if trueIndex == falseIndex {
			nextIndex = trueIndex
		}

	case *lockStatement:
		stmt.index = nextIndex
		nextIndex++

	case *unlockStatement:
		nextIndex++
	}

	return nextIndex
}

func typeCheckClause(contract *Contract, clause *Clause, env *environ) error {
	for _, s := range clause.statements {
		if err := typeCheckStatement(s, contract.Value, clause.Name, env); err != nil {
			return err
		}
	}
	return nil
}

func typeCheckStatement(stat statement, contractValue ValueInfo, clauseName string, env *environ) error {
	switch stmt := stat.(type) {
	case *ifStatement:
		for _, trueStmt := range stmt.body.trueBody {
			if err := typeCheckStatement(trueStmt, contractValue, clauseName, env); err != nil {
				return err
			}
		}

		for _, falseStmt := range stmt.body.falseBody {
			if err := typeCheckStatement(falseStmt, contractValue, clauseName, env); err != nil {
				return err
			}
		}

	case *defineStatement:
		if stmt.expr != nil && stmt.expr.typ(env) != stmt.variable.Type && !isHashSubtype(stmt.expr.typ(env)) {
			return fmt.Errorf("expression in define statement in clause \"%s\" has type \"%s\", must be \"%s\"",
				clauseName, stmt.expr.typ(env), stmt.variable.Type)
		}

	case *assignStatement:
		if stmt.expr.typ(env) != stmt.variable.Type && !isHashSubtype(stmt.expr.typ(env)) {
			return fmt.Errorf("expression in assign statement in clause \"%s\" has type \"%s\", must be \"%s\"",
				clauseName, stmt.expr.typ(env), stmt.variable.Type)
		}

	case *verifyStatement:
		if t := stmt.expr.typ(env); t != boolType {
			return fmt.Errorf("expression in verify statement in clause \"%s\" has type \"%s\", must be Boolean", clauseName, t)
		}

	case *lockStatement:
		if t := stmt.lockedAmount.typ(env); !(t == intType || t == amountType) {
			return fmt.Errorf("lockedAmount expression \"%s\" in lock statement in clause \"%s\" has type \"%s\", must be Integer", stmt.lockedAmount, clauseName, t)
		}
		if t := stmt.lockedAsset.typ(env); t != assetType {
			return fmt.Errorf("lockedAsset expression \"%s\" in lock statement in clause \"%s\" has type \"%s\", must be Asset", stmt.lockedAsset, clauseName, t)
		}
		if t := stmt.program.typ(env); t != progType {
			return fmt.Errorf("program in lock statement in clause \"%s\" has type \"%s\", must be Program", clauseName, t)
		}

	case *unlockStatement:
		if t := stmt.unlockedAmount.typ(env); !(t == intType || t == amountType) {
			return fmt.Errorf("unlockedAmount expression \"%s\" in unlock statement of clause \"%s\" has type \"%s\", must be Integer", stmt.unlockedAmount, clauseName, t)
		}
		if t := stmt.unlockedAsset.typ(env); t != assetType {
			return fmt.Errorf("unlockedAsset expression \"%s\" in unlock statement of clause \"%s\" has type \"%s\", must be Asset", stmt.unlockedAsset, clauseName, t)
		}
		if stmt.unlockedAmount.String() != contractValue.Amount || stmt.unlockedAsset.String() != contractValue.Asset {
			return fmt.Errorf("amount \"%s\" of asset \"%s\" expression in unlock statement of clause \"%s\" must be the contract valueAmount \"%s\" of valueAsset \"%s\"",
				stmt.unlockedAmount.String(), stmt.unlockedAsset.String(), clauseName, contractValue.Amount, contractValue.Asset)
		}
	}

	return nil
}
