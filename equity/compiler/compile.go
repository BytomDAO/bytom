package compiler

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/protocol/vm/vmutil"
)

// ValueInfo describes how a blockchain value is used in a contract
// clause.
type ValueInfo struct {
	// Name is the clause's name for this value.
	Name string `json:"name"`

	// Program is the program expression used to the lock the value, if
	// the value is locked with "lock." If it's unlocked with "unlock"
	// instead, this is empty.
	Program string `json:"program,omitempty"`

	// Asset is the expression describing the asset type the value must
	// have, as it appears in a clause's "requires" section. If this is
	// the contract value instead, this is empty.
	Asset string `json:"asset,omitempty"`

	// Amount is the expression describing the amount the value must
	// have, as it appears in a clause's "requires" section. If this is
	// the contract value instead, this is empty.
	Amount string `json:"amount,omitempty"`
}

// ContractArg is an argument with which to instantiate a contract as
// a program. Exactly one of B, I, and S should be supplied.
type ContractArg struct {
	B *bool               `json:"boolean,omitempty"`
	I *int64              `json:"integer,omitempty"`
	S *chainjson.HexBytes `json:"string,omitempty"`
}

// Compile parses a sequence of Equity contracts from the supplied reader
// and produces Contract objects containing the compiled bytecode and
// other analysis. If argMap is non-nil, it maps contract names to
// lists of arguments with which to instantiate them as programs, with
// the results placed in the contract's Program field. A contract
// named in argMap but not found in the input is silently ignored.
func Compile(r io.Reader) ([]*Contract, error) {
	inp, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "reading input")
	}
	contracts, err := parse(inp)
	if err != nil {
		return nil, errors.Wrap(err, "parse error")
	}

	globalEnv := newEnviron(nil)
	for _, k := range keywords {
		globalEnv.add(k, nilType, roleKeyword)
	}
	for _, b := range builtins {
		globalEnv.add(b.name, nilType, roleBuiltin)
	}

	// All contracts must be checked for recursiveness before any are
	// compiled.
	for _, contract := range contracts {
		contract.Recursive = checkRecursive(contract)
	}

	for _, contract := range contracts {
		err = globalEnv.addContract(contract)
		if err != nil {
			return nil, err
		}
	}

	for _, contract := range contracts {
		err = compileContract(contract, globalEnv)
		if err != nil {
			return nil, errors.Wrap(err, "compiling contract")
		}
		for _, clause := range contract.Clauses {
			for _, stmt := range clause.statements {
				switch s := stmt.(type) {
				case *lockStatement:
					valueInfo := ValueInfo{
						Amount:  s.lockedAmount.String(),
						Asset:   s.lockedAsset.String(),
						Program: s.program.String(),
					}

					clause.Values = append(clause.Values, valueInfo)
				case *unlockStatement:
					valueInfo := ValueInfo{
						Amount: contract.Value.Amount,
						Asset:  contract.Value.Asset,
					}
					clause.Values = append(clause.Values, valueInfo)
				}
			}
		}
	}

	return contracts, nil
}

func Instantiate(body []byte, params []*Param, recursive bool, args []ContractArg) ([]byte, error) {
	if len(args) != len(params) {
		return nil, fmt.Errorf("got %d argument(s), want %d", len(args), len(params))
	}

	// typecheck args against param types
	for i, param := range params {
		arg := args[i]
		switch param.Type {
		case amountType, intType:
			if arg.I == nil {
				return nil, fmt.Errorf("type mismatch in arg %d (want integer)", i)
			}
		case assetType, hashType, progType, pubkeyType, sigType, strType:
			if arg.S == nil {
				return nil, fmt.Errorf("type mismatch in arg %d (want string)", i)
			}
		case boolType:
			if arg.B == nil {
				return nil, fmt.Errorf("type mismatch in arg %d (want boolean)", i)
			}
		}
	}

	b := vmutil.NewBuilder()

	for i := len(args) - 1; i >= 0; i-- {
		a := args[i]
		switch {
		case a.B != nil:
			var n int64
			if *a.B {
				n = 1
			}
			b.AddInt64(n)
		case a.I != nil:
			b.AddInt64(*a.I)
		case a.S != nil:
			b.AddData(*a.S)
		}
	}

	if recursive {
		// <argN> <argN-1> ... <arg1> <body> DEPTH OVER 0 CHECKPREDICATE
		b.AddData(body)
		b.AddOp(vm.OP_DEPTH).AddOp(vm.OP_OVER)
	} else {
		// <argN> <argN-1> ... <arg1> DEPTH <body> 0 CHECKPREDICATE
		b.AddOp(vm.OP_DEPTH)
		b.AddData(body)
	}
	b.AddInt64(0)
	b.AddOp(vm.OP_CHECKPREDICATE)
	return b.Build()
}

func compileContract(contract *Contract, globalEnv *environ) error {
	var err error

	if len(contract.Clauses) == 0 {
		return fmt.Errorf("empty contract")
	}
	env := newEnviron(globalEnv)
	for _, p := range contract.Params {
		err = env.add(p.Name, p.Type, roleContractParam)
		if err != nil {
			return err
		}
	}

	// value is spilt with valueAmount and valueAsset
	if err = env.add(contract.Value.Amount, amountType, roleContractValue); err != nil {
		return err
	}
	if err = env.add(contract.Value.Asset, assetType, roleContractValue); err != nil {
		return err
	}

	for _, c := range contract.Clauses {
		err = env.add(c.Name, nilType, roleClause)
		if err != nil {
			return err
		}
	}

	err = prohibitSigParams(contract)
	if err != nil {
		return err
	}
	err = requireAllParamsUsedInClauses(contract.Params, contract.Clauses)
	if err != nil {
		return err
	}

	var stk stack

	if len(contract.Clauses) > 1 {
		stk = stk.add("<clause selector>")
	}

	for i := len(contract.Params) - 1; i >= 0; i-- {
		p := contract.Params[i]
		stk = stk.add(p.Name)
	}

	if contract.Recursive {
		stk = stk.add(contract.Name)
	}

	b := &builder{}
	sequence := 0 // sequence is used to count the number of ifStatements

	if len(contract.Clauses) == 1 {
		err = compileClause(b, stk, contract, env, contract.Clauses[0], &sequence)
		if err != nil {
			return err
		}
	} else {
		if len(contract.Params) > 0 {
			// A clause selector is at the bottom of the stack. Roll it to the
			// top.
			n := len(contract.Params)
			if contract.Recursive {
				n++
			}
			stk = b.addRoll(stk, n) // stack: [<clause params> <contract params> [<maybe contract body>] <clause selector>]
		}

		var stk2 stack

		// clauses 2..N-1
		for i := len(contract.Clauses) - 1; i >= 2; i-- {
			stk = b.addDup(stk)                                                   // stack: [... <clause selector> <clause selector>]
			stk = b.addInt64(stk, int64(i))                                       // stack: [... <clause selector> <clause selector> <i>]
			stk = b.addNumEqual(stk, fmt.Sprintf("(<clause selector> == %d)", i)) // stack: [... <clause selector> <i == clause selector>]
			stk = b.addJumpIf(stk, contract.Clauses[i].Name)                      // stack: [... <clause selector>]
			stk2 = stk                                                            // stack starts here for clauses 2 through N-1
		}

		// clause 1
		stk = b.addJumpIf(stk, contract.Clauses[1].Name) // consumes the clause selector

		// no jump needed for clause 0

		for i, clause := range contract.Clauses {
			if i > 1 {
				// Clauses 0 and 1 have no clause selector on top of the
				// stack. Clauses 2 and later do.
				stk = stk2
			}

			b.addJumpTarget(stk, clause.Name)

			if i > 1 {
				stk = b.addDrop(stk)
			}

			err = compileClause(b, stk, contract, env, clause, &sequence)
			if err != nil {
				return errors.Wrapf(err, "compiling clause \"%s\"", clause.Name)
			}
			b.forgetPendingVerify()
			if i < len(contract.Clauses)-1 {
				b.addJump(stk, "_end")
			}
		}
		b.addJumpTarget(stk, "_end")
	}

	opcodes := optimize(b.opcodes())
	prog, err := vm.Assemble(opcodes)
	if err != nil {
		return err
	}

	contract.Body = prog
	contract.Opcodes = opcodes

	contract.Steps = b.steps()

	return nil
}

func compileClause(b *builder, contractStk stack, contract *Contract, env *environ, clause *Clause, sequence *int) error {
	var err error

	// copy env to leave outerEnv unchanged
	env = newEnviron(env)
	for _, p := range clause.Params {
		err = env.add(p.Name, p.Type, roleClauseParam)
		if err != nil {
			return err
		}
	}

	if err = assignIndexes(clause); err != nil {
		return err
	}

	var stk stack
	for _, p := range clause.Params {
		// NOTE: the order of clause params is not reversed, unlike
		// contract params (and also unlike the arguments to Equity
		// function-calls).
		stk = stk.add(p.Name)
	}
	stk = stk.addFromStack(contractStk)

	// a count of the number of times each variable is referenced
	counts := make(map[string]int)
	for _, s := range clause.statements {
		if stmt, ok := s.(*defineStatement); ok && stmt.expr == nil {
			continue
		}

		s.countVarRefs(counts)
		if stmt, ok := s.(*ifStatement); ok {
			for _, trueStmt := range stmt.body.trueBody {
				trueStmt.countVarRefs(counts)
			}

			for _, falseStmt := range stmt.body.falseBody {
				falseStmt.countVarRefs(counts)
			}
		}
	}

	for _, stat := range clause.statements {
		if stk, err = compileStatement(b, stk, contract, env, clause, counts, stat, sequence); err != nil {
			return err
		}
	}

	err = typeCheckClause(contract, clause, env)
	if err != nil {
		return err
	}
	err = requireAllParamsUsedInClause(clause.Params, clause)
	if err != nil {
		return err
	}

	return nil
}

func compileStatement(b *builder, stk stack, contract *Contract, env *environ, clause *Clause, counts map[string]int, stat statement, sequence *int) (stack, error) {
	var err error
	switch stmt := stat.(type) {
	case *ifStatement:
		// sequence add 1 when the statement is ifStatement
		*sequence++
		strSequence := fmt.Sprintf("%d", *sequence)

		// compile condition expression
		stk, err = compileExpr(b, stk, contract, clause, env, counts, stmt.condition)
		if err != nil {
			return stk, errors.Wrapf(err, "in check condition of ifStatement in clause \"%s\"", clause.Name)
		}

		// jump to falseBody when condition is false, while the JUMPIF instruction will be run success when
		// the value of dataStack is true, therefore add this negation
		conditionExpr := stk.str
		stk = b.addNot(stk, fmt.Sprintf("!%s", conditionExpr))

		// add nop instruction to differ with clause selector for JUMPIF instruction
		stk = b.addNop(stk)

		// add label
		var label string
		if len(stmt.body.falseBody) != 0 {
			label = "else_" + strSequence
		} else {
			label = "endif_" + strSequence
		}
		stk = b.addJumpIf(stk, label)
		b.addJumpTarget(stk, "if_"+strSequence)

		// temporary store stack and counts for falseBody
		condStk := stk
		elseCounts := make(map[string]int)
		for k, v := range counts {
			elseCounts[k] = v
		}

		// compile trueBody statements
		if len(stmt.body.trueBody) != 0 {
			for _, st := range stmt.body.trueBody {
				st.countVarRefs(counts)
			}

			for _, st := range stmt.body.trueBody {
				if stk, err = compileStatement(b, stk, contract, env, clause, counts, st, sequence); err != nil {
					return stk, err
				}
			}
		}

		// compile falseBody statements
		if len(stmt.body.falseBody) != 0 {
			counts := make(map[string]int)
			for k, v := range elseCounts {
				counts[k] = v
			}

			for _, st := range stmt.body.falseBody {
				st.countVarRefs(counts)
			}

			stk = condStk
			b.addJump(stk, "endif_"+strSequence)
			b.addJumpTarget(stk, "else_"+strSequence)

			for _, st := range stmt.body.falseBody {
				if stk, err = compileStatement(b, stk, contract, env, clause, counts, st, sequence); err != nil {
					return stk, err
				}
			}
		}
		b.addJumpTarget(stk, "endif_"+strSequence)

	case *defineStatement:
		// add environ for define variable
		if err = env.add(stmt.variable.Name, stmt.variable.Type, roleClauseVariable); err != nil {
			return stk, err
		}

		// check whether the variable is used or not
		if counts[stmt.variable.Name] == 0 {
			return stk, fmt.Errorf("the defined variable \"%s\" is unused in clause \"%s\"", stmt.variable.Name, clause.Name)
		}

		if stmt.expr != nil {
			// variable
			stk, err = compileExpr(b, stk, contract, clause, env, counts, stmt.expr)
			if err != nil {
				return stk, errors.Wrapf(err, "in define statement in clause \"%s\"", clause.Name)
			}

			// modify stack name
			stk.str = stmt.variable.Name
		}

	case *assignStatement:
		// find variable from environ with roleClauseVariable
		if entry := env.lookup(string(stmt.variable.Name)); entry != nil {
			if entry.r != roleClauseVariable {
				return stk, fmt.Errorf("the type of variable is not roleClauseVariable in assign statement in clause \"%s\"", clause.Name)
			}
			stmt.variable.Type = entry.t
		} else {
			return stk, fmt.Errorf("the variable \"%s\" is not defined before the assign statement in clause \"%s\"", stmt.variable.Name, clause.Name)
		}

		// temporary store the counts of defined variable
		varCount := counts[stmt.variable.Name]

		// calculate the counts of variable for assign statement
		tmpCounts := make(map[string]int)
		stmt.countVarRefs(tmpCounts)

		// modify the map counts of defined variable to 1 and minus the number of defined variable
		// when the assign expression contains the defined variable
		if tmpCounts[stmt.variable.Name] > 0 {
			counts[stmt.variable.Name] = 1
			varCount -= tmpCounts[stmt.variable.Name]
		} else {
			depth := stk.find(stmt.variable.Name)
			switch depth {
			case 0:
				break
			case 1:
				stk = b.addSwap(stk)
			default:
				stk = b.addRoll(stk, depth)
			}
			stk = b.addDrop(stk)
		}

		// variable
		stk, err = compileExpr(b, stk, contract, clause, env, counts, stmt.expr)
		if err != nil {
			return stk, errors.Wrapf(err, "in define statement in clause \"%s\"", clause.Name)
		}

		// restore the defined variable counts
		if tmpCounts[stmt.variable.Name] > 0 {
			counts[stmt.variable.Name] = varCount
		}

		// modify stack name
		stk.str = stmt.variable.Name

	case *verifyStatement:
		stk, err = compileExpr(b, stk, contract, clause, env, counts, stmt.expr)
		if err != nil {
			return stk, errors.Wrapf(err, "in verify statement in clause \"%s\"", clause.Name)
		}
		stk = b.addVerify(stk)

		// special-case reporting of certain function calls
		if c, ok := stmt.expr.(*callExpr); ok && len(c.args) == 1 {
			if b := referencedBuiltin(c.fn); b != nil {
				switch b.name {
				case "below":
					clause.BlockHeight = append(clause.BlockHeight, c.args[0].String())
				case "above":
					clause.BlockHeight = append(clause.BlockHeight, c.args[0].String())
				}
			}
		}

	case *lockStatement:
		// index
		stk = b.addInt64(stk, stmt.index)

		// TODO: permit more complex expressions for locked,
		// like "lock x+y with foo" (?)

		if stmt.lockedAmount.String() == contract.Value.Amount && stmt.lockedAsset.String() == contract.Value.Asset {
			stk = b.addAmount(stk, contract.Value.Amount)
			stk = b.addAsset(stk, contract.Value.Asset)
		} else {
			// calculate the counts of variable for lockStatement
			lockCounts := make(map[string]int)
			stmt.countVarRefs(lockCounts)

			// amount
			switch {
			case stmt.lockedAmount.String() == contract.Value.Amount:
				stk = b.addAmount(stk, contract.Value.Amount)
			case stmt.lockedAmount.String() != contract.Value.Amount && lockCounts[contract.Value.Amount] > 0:
				counts[contract.Value.Amount] = lockCounts[contract.Value.Amount]
				stk = b.addAmount(stk, contract.Value.Amount)
				stk, err = compileExpr(b, stk, contract, clause, env, counts, stmt.lockedAmount)
				if err != nil {
					return stk, errors.Wrapf(err, "in lock statement in clause \"%s\"", clause.Name)
				}
			default:
				stk, err = compileExpr(b, stk, contract, clause, env, counts, stmt.lockedAmount)
				if err != nil {
					return stk, errors.Wrapf(err, "in lock statement in clause \"%s\"", clause.Name)
				}
			}

			// asset
			switch {
			case stmt.lockedAsset.String() == contract.Value.Asset:
				stk = b.addAsset(stk, contract.Value.Asset)
			case stmt.lockedAsset.String() != contract.Value.Asset && lockCounts[contract.Value.Asset] > 0:
				counts[contract.Value.Asset] = lockCounts[contract.Value.Asset]
				stk = b.addAsset(stk, contract.Value.Asset)
				stk, err = compileExpr(b, stk, contract, clause, env, counts, stmt.lockedAsset)
				if err != nil {
					return stk, errors.Wrapf(err, "in lock statement in clause \"%s\"", clause.Name)
				}
			default:
				stk, err = compileExpr(b, stk, contract, clause, env, counts, stmt.lockedAsset)
				if err != nil {
					return stk, errors.Wrapf(err, "in lock statement in clause \"%s\"", clause.Name)
				}
			}
		}

		// version
		stk = b.addInt64(stk, 1)

		// prog
		stk, err = compileExpr(b, stk, contract, clause, env, counts, stmt.program)
		if err != nil {
			return stk, errors.Wrapf(err, "in lock statement in clause \"%s\"", clause.Name)
		}

		stk = b.addCheckOutput(stk, fmt.Sprintf("checkOutput(%s, %s, %s)",
			stmt.lockedAmount.String(), stmt.lockedAsset.String(), stmt.program))
		stk = b.addVerify(stk)

	case *unlockStatement:
		if len(clause.statements) == 1 {
			// This is the only statement in the clause, make sure TRUE is
			// on the stack.
			stk = b.addBoolean(stk, true)
		}
	}

	return stk, nil
}

func compileExpr(b *builder, stk stack, contract *Contract, clause *Clause, env *environ, counts map[string]int, expr expression) (stack, error) {
	var err error

	switch e := expr.(type) {
	case *binaryExpr:
		// Do typechecking after compiling subexpressions (because other
		// compilation errors are more interesting than type mismatch
		// errors).

		stk, err = compileExpr(b, stk, contract, clause, env, counts, e.left)
		if err != nil {
			return stk, errors.Wrapf(err, "in left operand of \"%s\" expression", e.op.op)
		}
		stk, err = compileExpr(b, stk, contract, clause, env, counts, e.right)
		if err != nil {
			return stk, errors.Wrapf(err, "in right operand of \"%s\" expression", e.op.op)
		}

		lType := e.left.typ(env)
		if e.op.left != "" && ((e.op.left == intType && !(lType == amountType || lType == intType)) ||
			(e.op.left == boolType && !(lType == boolType))) {
			return stk, fmt.Errorf("in \"%s\", left operand has type \"%s\", must be \"%s\"", e, lType, e.op.left)
		}

		rType := e.right.typ(env)
		if e.op.right != "" && ((e.op.right == intType && !(rType == amountType || rType == intType)) ||
			(e.op.right == boolType && !(rType == boolType))) {
			return stk, fmt.Errorf("in \"%s\", right operand has type \"%s\", must be \"%s\"", e, rType, e.op.right)
		}

		switch e.op.op {
		case "==", "!=":
			if lType != rType {
				// Maybe one is Hash and the other is (more-specific-Hash subtype).
				// TODO(bobg): generalize this mechanism
				if lType == hashType && isHashSubtype(rType) {
					propagateType(contract, clause, env, rType, e.left)
				} else if rType == hashType && isHashSubtype(lType) {
					propagateType(contract, clause, env, lType, e.right)
				} else {
					return stk, fmt.Errorf("type mismatch in \"%s\": left operand has type \"%s\", right operand has type \"%s\"", e, lType, rType)
				}
			}
			if lType == "Boolean" {
				return stk, fmt.Errorf("in \"%s\": using \"%s\" on Boolean values not allowed", e, e.op.op)
			}
		}

		stk = b.addOps(stk.dropN(2), e.op.opcodes, e.String())

	case *unaryExpr:
		// Do typechecking after compiling subexpression (because other
		// compilation errors are more interesting than type mismatch
		// errors).

		var err error
		stk, err = compileExpr(b, stk, contract, clause, env, counts, e.expr)
		if err != nil {
			return stk, errors.Wrapf(err, "in \"%s\" expression", e.op.op)
		}

		if e.op.operand != "" && e.expr.typ(env) != e.op.operand {
			return stk, fmt.Errorf("in \"%s\", operand has type \"%s\", must be \"%s\"", e, e.expr.typ(env), e.op.operand)
		}
		b.addOps(stk.drop(), e.op.opcodes, e.String())

	case *callExpr:
		bi := referencedBuiltin(e.fn)
		if bi == nil {
			if v, ok := e.fn.(varRef); ok {
				if entry := env.lookup(string(v)); entry != nil && entry.t == contractType {
					clause.Contracts = append(clause.Contracts, entry.c.Name)

					partialName := fmt.Sprintf("%s(...)", v)
					stk = b.addData(stk, nil)

					if len(e.args) != len(entry.c.Params) {
						return stk, fmt.Errorf("contract \"%s\" expects %d argument(s), got %d", entry.c.Name, len(entry.c.Params), len(e.args))
					}

					for i := len(e.args) - 1; i >= 0; i-- {
						arg := e.args[i]
						if entry.c.Params[i].Type != "" && arg.typ(env) != entry.c.Params[i].Type &&
							!(arg.typ(env) == intType && entry.c.Params[i].Type == amountType) {
							return stk, fmt.Errorf("argument %d to contract \"%s\" has type \"%s\", must be \"%s\"", i, entry.c.Name, arg.typ(env), entry.c.Params[i].Type)
						}
						stk, err = compileExpr(b, stk, contract, clause, env, counts, arg)
						if err != nil {
							return stk, err
						}
						stk = b.addCatPushdata(stk, partialName)
					}

					switch {
					case entry.c == contract:
						// Recursive call - cannot use entry.c.Body
						// <argN> <argN-1> ... <arg1> <body> DEPTH OVER 0 CHECKPREDICATE
						stk, err = compileRef(b, stk, counts, varRef(contract.Name))
						if err != nil {
							return stk, errors.Wrap(err, "compiling contract call")
						}
						stk = b.addCatPushdata(stk, partialName)
						stk = b.addData(stk, []byte{byte(vm.OP_DEPTH), byte(vm.OP_OVER)})
						stk = b.addCat(stk, partialName)

					case entry.c.Recursive:
						// Non-recursive call to a (different) recursive contract
						// <argN> <argN-1> ... <arg1> <body> DEPTH OVER 0 CHECKPREDICATE
						if len(entry.c.Body) == 0 {
							// TODO(bobg): sort input contracts topologically to permit forward calling
							return stk, fmt.Errorf("contract \"%s\" not defined", entry.c.Name)
						}
						stk = b.addData(stk, entry.c.Body)
						stk = b.addCatPushdata(stk, partialName)
						stk = b.addData(stk, []byte{byte(vm.OP_DEPTH), byte(vm.OP_OVER)})
						stk = b.addCat(stk, partialName)

					default:
						// Non-recursive call to non-recursive contract
						// <argN> <argN-1> ... <arg1> DEPTH <body> 0 CHECKPREDICATE
						stk = b.addData(stk, []byte{byte(vm.OP_DEPTH)})
						stk = b.addCat(stk, partialName)
						if len(entry.c.Body) == 0 {
							// TODO(bobg): sort input contracts topologically to permit forward calling
							return stk, fmt.Errorf("contract \"%s\" not defined", entry.c.Name)
						}
						stk = b.addData(stk, entry.c.Body)
						stk = b.addCatPushdata(stk, partialName)
					}
					stk = b.addData(stk, vm.Int64Bytes(0))
					stk = b.addCatPushdata(stk, partialName)
					stk = b.addData(stk, []byte{byte(vm.OP_CHECKPREDICATE)})
					stk = b.addCat(stk, e.String())

					return stk, nil
				}
			}
			return stk, fmt.Errorf("unknown function \"%s\"", e.fn)
		}

		if len(e.args) != len(bi.args) {
			return stk, fmt.Errorf("wrong number of args for \"%s\": have %d, want %d", bi.name, len(e.args), len(bi.args))
		}

		// WARNING WARNING WOOP WOOP
		// special-case hack
		// WARNING WARNING WOOP WOOP
		if bi.name == "checkTxMultiSig" {
			if _, ok := e.args[0].(listExpr); !ok {
				return stk, fmt.Errorf("checkTxMultiSig expects list literals, got %T for argument 0", e.args[0])
			}
			if _, ok := e.args[1].(listExpr); !ok {
				return stk, fmt.Errorf("checkTxMultiSig expects list literals, got %T for argument 1", e.args[1])
			}

			var k1, k2 int

			stk, k1, err = compileArg(b, stk, contract, clause, env, counts, e.args[1])
			if err != nil {
				return stk, err
			}

			// stack: [... sigM ... sig1 M]

			var altEntry string
			stk, altEntry = b.addToAltStack(stk) // stack: [... sigM ... sig1]
			stk = b.addTxSigHash(stk)            // stack: [... sigM ... sig1 txsighash]

			stk, k2, err = compileArg(b, stk, contract, clause, env, counts, e.args[0])
			if err != nil {
				return stk, err
			}

			// stack: [... sigM ... sig1 txsighash pubkeyN ... pubkey1 N]

			stk = b.addFromAltStack(stk, altEntry) // stack: [... sigM ... sig1 txsighash pubkeyN ... pubkey1 N M]
			stk = b.addSwap(stk)                   // stack: [... sigM ... sig1 txsighash pubkeyN ... pubkey1 M N]
			stk = b.addCheckMultisig(stk, k1+k2, e.String())

			return stk, nil
		}

		var k int

		for i := len(e.args) - 1; i >= 0; i-- {
			a := e.args[i]
			var k2 int
			var err error
			stk, k2, err = compileArg(b, stk, contract, clause, env, counts, a)
			if err != nil {
				return stk, errors.Wrapf(err, "compiling argument %d in call expression", i)
			}
			k += k2
		}

		// Do typechecking after compiling subexpressions (because other
		// compilation errors are more interesting than type mismatch
		// errors).
		for i, actual := range e.args {
			if bi.args[i] != "" && actual.typ(env) != bi.args[i] {
				return stk, fmt.Errorf("argument %d to \"%s\" has type \"%s\", must be \"%s\"", i, bi.name, actual.typ(env), bi.args[i])
			}
		}

		stk = b.addOps(stk.dropN(k), bi.opcodes, e.String())

		// special-case reporting
		switch bi.name {
		case "sha3", "sha256":
			clause.HashCalls = append(clause.HashCalls, HashCall{bi.name, e.args[0].String(), string(e.args[0].typ(env))})
		}

	case varRef:
		return compileRef(b, stk, counts, e)

	case integerLiteral:
		stk = b.addInt64(stk, int64(e))

	case bytesLiteral:
		stk = b.addData(stk, []byte(e))

	case booleanLiteral:
		stk = b.addBoolean(stk, bool(e))

	case listExpr:
		// Lists are excluded here because they disobey the invariant of
		// this function: namely, that it increases the stack size by
		// exactly one. (A list pushes its items and its length on the
		// stack.) But they're OK as function-call arguments because the
		// function (presumably) consumes all the stack items added.
		return stk, fmt.Errorf("encountered list outside of function-call context")
	}
	return stk, nil
}

func compileArg(b *builder, stk stack, contract *Contract, clause *Clause, env *environ, counts map[string]int, expr expression) (stack, int, error) {
	var n int
	if list, ok := expr.(listExpr); ok {
		for i := 0; i < len(list); i++ {
			elt := list[len(list)-i-1]
			var err error
			stk, err = compileExpr(b, stk, contract, clause, env, counts, elt)
			if err != nil {
				return stk, 0, err
			}
			n++
		}
		stk = b.addInt64(stk, int64(len(list)))
		n++
		return stk, n, nil
	}
	var err error
	stk, err = compileExpr(b, stk, contract, clause, env, counts, expr)
	return stk, 1, err
}

func compileRef(b *builder, stk stack, counts map[string]int, ref varRef) (stack, error) {
	depth := stk.find(string(ref))
	if depth < 0 {
		return stk, fmt.Errorf("undefined reference: \"%s\"", ref)
	}

	var isFinal bool
	if count, ok := counts[string(ref)]; ok && count > 0 {
		count--
		counts[string(ref)] = count
		isFinal = count == 0
	}

	switch depth {
	case 0:
		if !isFinal {
			stk = b.addDup(stk)
		}
	case 1:
		if isFinal {
			stk = b.addSwap(stk)
		} else {
			stk = b.addOver(stk)
		}
	default:
		if isFinal {
			stk = b.addRoll(stk, depth)
		} else {
			stk = b.addPick(stk, depth)
		}
	}
	return stk, nil
}

func (a *ContractArg) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	if r, ok := m["boolean"]; ok {
		var bval bool
		err = json.Unmarshal(r, &bval)
		if err != nil {
			return err
		}
		a.B = &bval
		return nil
	}
	if r, ok := m["integer"]; ok {
		var ival int64
		err = json.Unmarshal(r, &ival)
		if err != nil {
			return err
		}
		a.I = &ival
		return nil
	}
	r, ok := m["string"]
	if !ok {
		return fmt.Errorf("contract arg must define one of boolean, integer, string")
	}
	var sval chainjson.HexBytes
	err = json.Unmarshal(r, &sval)
	if err != nil {
		return err
	}
	a.S = &sval
	return nil
}
