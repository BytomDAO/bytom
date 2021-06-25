package vm

import (
	"encoding/binary"
)

func opVerify(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	p, err := vm.pop(true)
	if err != nil {
		return err
	}

	if AsBool(p) {
		return nil
	}
	return ErrVerifyFailed
}

func opFail(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	return ErrReturn
}

func opCheckPredicate(vm *virtualMachine) error {
	if err := vm.applyCost(256); err != nil {
		return err
	}

	vm.deferCost(-256 + 64) // get most of that cost back at the end
	limitBigInt, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	limit, err := bigIntInt64(limitBigInt)
	if err != nil {
		return err
	}

	predicate, err := vm.pop(true)
	if err != nil {
		return err
	}

	nBigInt, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	n, err := bigIntInt64(nBigInt)
	if err != nil {
		return err
	}

	l := int64(len(vm.dataStack))
	if n == 0 {
		n = l
	}
	if n > l {
		return ErrDataStackUnderflow
	}
	if limit == 0 {
		limit = vm.runLimit
	}

	if err = vm.applyCost(limit); err != nil {
		return err
	}

	childVM := virtualMachine{
		context:   vm.context,
		program:   predicate,
		runLimit:  limit,
		depth:     vm.depth + 1,
		dataStack: append([][]byte{}, vm.dataStack[l-n:]...),
	}
	vm.dataStack = vm.dataStack[:l-n]

	childErr := childVM.run()

	vm.deferCost(-childVM.runLimit)
	vm.deferCost(-stackCost(childVM.dataStack))
	vm.deferCost(-stackCost(childVM.altStack))

	return vm.pushBool(childErr == nil && !childVM.falseResult(), true)
}

func opJump(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	address := binary.LittleEndian.Uint32(vm.data)
	vm.nextPC = address
	return nil
}

func opJumpIf(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	p, err := vm.pop(true)
	if err != nil {
		return err
	}

	if AsBool(p) {
		address := binary.LittleEndian.Uint32(vm.data)
		vm.nextPC = address
	}
	return nil
}
