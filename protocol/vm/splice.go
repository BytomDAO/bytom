package vm

import (
	"github.com/holiman/uint256"

	"github.com/bytom/bytom/math/checked"
)

func opCat(vm *virtualMachine) error {
	err := vm.applyCost(4)
	if err != nil {
		return err
	}
	b, err := vm.pop(true)
	if err != nil {
		return err
	}
	a, err := vm.pop(true)
	if err != nil {
		return err
	}
	lens := int64(len(a) + len(b))
	err = vm.applyCost(lens)
	if err != nil {
		return err
	}
	vm.deferCost(-lens)
	err = vm.push(append(a, b...), true)
	if err != nil {
		return err
	}
	return nil
}

func opSubstr(vm *virtualMachine) error {
	if err := vm.applyCost(4); err != nil {
		return err
	}

	sizeBigInt, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	size, err := bigIntInt64(sizeBigInt)
	if err != nil {
		return err
	}

	if err = vm.applyCost(size); err != nil {
		return err
	}

	vm.deferCost(-size)
	offsetBigInt, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	offset, err := bigIntInt64(offsetBigInt)
	if err != nil {
		return err
	}

	str, err := vm.pop(true)
	if err != nil {
		return err
	}

	end, ok := checked.AddInt64(offset, size)
	if !ok || end > int64(len(str)) {
		return ErrBadValue
	}

	return vm.push(str[offset:end], true)
}

func opLeft(vm *virtualMachine) error {
	if err := vm.applyCost(4); err != nil {
		return err
	}

	sizeBigInt, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	size, err := bigIntInt64(sizeBigInt)
	if err != nil {
		return err
	}

	if err = vm.applyCost(size); err != nil {
		return err
	}

	vm.deferCost(-size)
	str, err := vm.pop(true)
	if err != nil {
		return err
	}

	if size > int64(len(str)) {
		return ErrBadValue
	}

	return vm.push(str[:size], true)
}

func opRight(vm *virtualMachine) error {
	if err := vm.applyCost(4); err != nil {
		return err
	}

	sizeBigInt, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	size, err := bigIntInt64(sizeBigInt)
	if err != nil {
		return err
	}

	if err = vm.applyCost(size); err != nil {
		return err
	}

	vm.deferCost(-size)
	str, err := vm.pop(true)
	if err != nil {
		return err
	}
	lstr := int64(len(str))
	if size > lstr {
		return ErrBadValue
	}

	return vm.push(str[lstr-size:], true)
}

func opSize(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	str, err := vm.top()
	if err != nil {
		return err
	}

	return vm.pushBigInt(uint256.NewInt().SetUint64(uint64(len(str))), true)
}

func opCatpushdata(vm *virtualMachine) error {
	err := vm.applyCost(4)
	if err != nil {
		return err
	}
	b, err := vm.pop(true)
	if err != nil {
		return err
	}
	a, err := vm.pop(true)
	if err != nil {
		return err
	}
	lb := len(b)
	lens := int64(len(a) + lb)
	err = vm.applyCost(lens)
	if err != nil {
		return err
	}
	vm.deferCost(-lens)
	return vm.push(append(a, PushDataBytes(b)...), true)
}
