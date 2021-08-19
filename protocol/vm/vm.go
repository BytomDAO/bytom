package vm

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/holiman/uint256"

	"github.com/bytom/bytom/errors"
)

type virtualMachine struct {
	context *Context

	program      []byte // the program currently executing
	pc, nextPC   uint32
	runLimit     int64
	deferredCost int64

	expansionReserved bool

	// Stores the data parsed out of an opcode. Used as input to
	// data-pushing opcodes.
	data []byte

	// CHECKPREDICATE spawns a child vm with depth+1
	depth int

	// In each of these stacks, stack[len(stack)-1] is the top element.
	dataStack [][]byte
	altStack  [][]byte
}

// TraceOut - if non-nil - will receive trace output during
// execution.
var TraceOut io.Writer

// Verify program by running VM
func Verify(context *Context, gasLimit int64) (gasLeft int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			if rErr, ok := r.(error); ok {
				err = errors.Sub(ErrUnexpected, rErr)
			} else {
				err = errors.Wrap(ErrUnexpected, r)
			}
		}
	}()

	if context.VMVersion != 1 {
		return gasLimit, ErrUnsupportedVM
	}

	vm := &virtualMachine{
		expansionReserved: context.TxVersion != nil && *context.TxVersion == 1,
		program:           context.Code,
		runLimit:          gasLimit,
		context:           context,
	}

	for i, state := range context.StateData {
		if err = vm.pushAltStack(state, false); err != nil {
			return vm.runLimit, errors.Wrapf(err, "pushing initial statedata %d", i)
		}
	}

	for i, arg := range context.Arguments {
		if err = vm.pushDataStack(arg, false); err != nil {
			return vm.runLimit, errors.Wrapf(err, "pushing initial argument %d", i)
		}
	}

	if err = vm.run(); err == nil && vm.falseResult() {
		err = ErrFalseVMResult
	}

	return vm.runLimit, wrapErr(err, vm, context.Arguments)
}

// falseResult returns true iff the stack is empty or the top
// item is false
func (vm *virtualMachine) falseResult() bool {
	return len(vm.dataStack) == 0 || !AsBool(vm.dataStack[len(vm.dataStack)-1])
}

func (vm *virtualMachine) run() error {
	for vm.pc = 0; vm.pc < uint32(len(vm.program)); { // handle vm.pc updates in step
		if err := vm.step(); err != nil {
			return err
		}
	}
	return nil
}

func (vm *virtualMachine) step() error {
	inst, err := ParseOp(vm.program, vm.pc)
	if err != nil {
		return err
	}

	vm.nextPC = vm.pc + inst.Len

	if TraceOut != nil {
		opname := inst.Op.String()
		fmt.Fprintf(TraceOut, "vm %d pc %d limit %d %s", vm.depth, vm.pc, vm.runLimit, opname)
		if len(inst.Data) > 0 {
			fmt.Fprintf(TraceOut, " %x", inst.Data)
		}
		fmt.Fprint(TraceOut, "\n")
	}

	if isExpansion[inst.Op] {
		if vm.expansionReserved {
			return ErrDisallowedOpcode
		}

		vm.pc = vm.nextPC
		return vm.applyCost(1)
	}

	vm.deferredCost = 0
	vm.data = inst.Data
	if err = ops[inst.Op].fn(vm); err != nil {
		return err
	}

	if err = vm.applyCost(vm.deferredCost); err != nil {
		return err
	}

	vm.pc = vm.nextPC
	if TraceOut != nil {
		for i := len(vm.dataStack) - 1; i >= 0; i-- {
			fmt.Fprintf(TraceOut, "  stack %d: %x\n", len(vm.dataStack)-1-i, vm.dataStack[i])
		}
	}

	return nil
}

func (vm *virtualMachine) pushDataStack(data []byte, deferred bool) error {
	cost := 8 + int64(len(data))
	if deferred {
		vm.deferCost(cost)
	} else if err := vm.applyCost(cost); err != nil {
		return err
	}

	vm.dataStack = append(vm.dataStack, data)
	return nil
}

func (vm *virtualMachine) pushAltStack(data []byte, deferred bool) error {
	cost := 8 + int64(len(data))
	if deferred {
		vm.deferCost(cost)
	} else if err := vm.applyCost(cost); err != nil {
		return err
	}

	vm.altStack = append(vm.altStack, data)
	return nil
}

func (vm *virtualMachine) pushBool(b bool, deferred bool) error {
	return vm.pushDataStack(BoolBytes(b), deferred)
}

func (vm *virtualMachine) pushBigInt(n *uint256.Int, deferred bool) error {
	return vm.pushDataStack(BigIntBytes(n), deferred)
}

func (vm *virtualMachine) pop(deferred bool) ([]byte, error) {
	if len(vm.dataStack) == 0 {
		return nil, ErrDataStackUnderflow
	}

	res := vm.dataStack[len(vm.dataStack)-1]
	vm.dataStack = vm.dataStack[:len(vm.dataStack)-1]

	cost := 8 + int64(len(res))
	if deferred {
		vm.deferCost(-cost)
	} else {
		vm.runLimit += cost
	}

	return res, nil
}

func (vm *virtualMachine) popBigInt(deferred bool) (*uint256.Int, error) {
	bytes, err := vm.pop(deferred)
	if err != nil {
		return nil, err
	}

	return AsBigInt(bytes)
}

func (vm *virtualMachine) top() ([]byte, error) {
	if len(vm.dataStack) == 0 {
		return nil, ErrDataStackUnderflow
	}

	return vm.dataStack[len(vm.dataStack)-1], nil
}

// positive cost decreases runlimit, negative cost increases it
func (vm *virtualMachine) applyCost(n int64) error {
	if n > vm.runLimit {
		vm.runLimit = 0
		return ErrRunLimitExceeded
	}

	vm.runLimit -= n
	return nil
}

func (vm *virtualMachine) deferCost(n int64) {
	vm.deferredCost += n
}

func stackCost(stack [][]byte) int64 {
	result := int64(8 * len(stack))
	for _, item := range stack {
		result += int64(len(item))
	}
	return result
}

func wrapErr(err error, vm *virtualMachine, args [][]byte) error {
	if err == nil {
		return nil
	}

	dis, errDis := Disassemble(vm.program)
	if errDis != nil {
		dis = "???"
	}

	dataArgs := make([]string, 0, len(args))
	for _, a := range args {
		dataArgs = append(dataArgs, hex.EncodeToString(a))
	}

	return errors.Wrap(err, fmt.Sprintf("%s [prog %x = %s; args %s]", err.Error(), vm.program, dis, strings.Join(dataArgs, " ")))
}
