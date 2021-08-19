package vm

import (
	"github.com/holiman/uint256"
)

func op1Add(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
		return err
	}

	n, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	num := uint256.NewInt(1)
	if num.Add(n, num); num.Sign() < 0 {
		return ErrRange
	}

	return vm.pushBigInt(num, true)
}

func op1Sub(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
		return err
	}

	n, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	num := uint256.NewInt(1)
	if num.Sub(n, num); num.Sign() < 0 {
		return ErrRange
	}

	return vm.pushBigInt(num, true)
}

func op2Mul(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
		return err
	}

	n, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	num := uint256.NewInt(2)
	if num.Mul(n, num); num.Sign() < 0 {
		return ErrRange
	}

	return vm.pushBigInt(num, true)
}

func op2Div(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}

	n, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	return vm.pushBigInt(n.Rsh(n, 1), true)
}

func opNot(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
		return err
	}

	n, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	return vm.pushBool(n.Cmp(uint256.NewInt(0)) == 0, true)
}

func op0NotEqual(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
		return err
	}

	n, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	return vm.pushBool(n.Cmp(uint256.NewInt(0)) != 0, true)
}

func opAdd(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
		return err
	}

	y, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	if x.Add(x, y); x.Sign() < 0 {
		return ErrRange
	}

	return vm.pushBigInt(x, true)
}

func opSub(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
		return err
	}

	y, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	if x.Sub(x, y); x.Sign() < 0 {
		return ErrRange
	}

	return vm.pushBigInt(x, true)
}

func opMul(vm *virtualMachine) error {
	if err := vm.applyCost(8); err != nil {
		return err
	}

	y, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	if _, overflow := x.MulOverflow(x, y); overflow || x.Sign() < 0 {
		return ErrRange
	}

	return vm.pushBigInt(x, true)
}

func opDiv(vm *virtualMachine) error {
	if err := vm.applyCost(8); err != nil {
		return err
	}

	y, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	if y.IsZero() {
		return ErrDivZero
	}

	return vm.pushBigInt(x.Div(x, y), true)
}

func opMod(vm *virtualMachine) error {
	if err := vm.applyCost(8); err != nil {
		return err
	}

	y, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	if y.IsZero() {
		return ErrDivZero
	}

	return vm.pushBigInt(x.Mod(x, y), true)
}

func opLshift(vm *virtualMachine) error {
	if err := vm.applyCost(8); err != nil {
		return err
	}

	y, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	if y.LtUint64(256) {
		x.Lsh(x, uint(y.Uint64()))
	} else {
		x.Clear()
	}

	if x.Sign() < 0 {
		return ErrRange
	}
	return vm.pushBigInt(x, true)
}

func opRshift(vm *virtualMachine) error {
	if err := vm.applyCost(8); err != nil {
		return err
	}

	y, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	if y.LtUint64(256) {
		x.Rsh(x, uint(y.Uint64()))
	} else {
		x.Clear()
	}

	return vm.pushBigInt(x, true)
}

func opBoolAnd(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
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
	return vm.pushBool(AsBool(a) && AsBool(b), true)
}

func opBoolOr(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
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
	return vm.pushBool(AsBool(a) || AsBool(b), true)
}

const (
	cmpLess = iota
	cmpLessEqual
	cmpGreater
	cmpGreaterEqual
	cmpEqual
	cmpNotEqual
)

func opNumEqual(vm *virtualMachine) error {
	return doNumCompare(vm, cmpEqual)
}

func opNumEqualVerify(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
		return err
	}

	y, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	if x.Eq(y) {
		return nil
	}
	return ErrVerifyFailed
}

func opNumNotEqual(vm *virtualMachine) error {
	return doNumCompare(vm, cmpNotEqual)
}

func opLessThan(vm *virtualMachine) error {
	return doNumCompare(vm, cmpLess)
}

func opGreaterThan(vm *virtualMachine) error {
	return doNumCompare(vm, cmpGreater)
}

func opLessThanOrEqual(vm *virtualMachine) error {
	return doNumCompare(vm, cmpLessEqual)
}

func opGreaterThanOrEqual(vm *virtualMachine) error {
	return doNumCompare(vm, cmpGreaterEqual)
}

func doNumCompare(vm *virtualMachine, op int) error {
	if err := vm.applyCost(2); err != nil {
		return err
	}

	y, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	var res bool
	switch op {
	case cmpLess:
		res = x.Cmp(y) < 0
	case cmpLessEqual:
		res = x.Cmp(y) <= 0
	case cmpGreater:
		res = x.Cmp(y) > 0
	case cmpGreaterEqual:
		res = x.Cmp(y) >= 0
	case cmpEqual:
		res = x.Cmp(y) == 0
	case cmpNotEqual:
		res = x.Cmp(y) != 0
	}
	return vm.pushBool(res, true)
}

func opMin(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
		return err
	}

	y, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	if x.Cmp(y) > 0 {
		return vm.pushBigInt(y, true)
	}
	return vm.pushBigInt(x, true)
}

func opMax(vm *virtualMachine) error {
	if err := vm.applyCost(2); err != nil {
		return err
	}

	y, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	if x.Cmp(y) < 0 {
		return vm.pushBigInt(y, true)
	}
	return vm.pushBigInt(x, true)
}

func opWithin(vm *virtualMachine) error {
	if err := vm.applyCost(4); err != nil {
		return err
	}

	max, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	min, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	x, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	return vm.pushBool(x.Cmp(min) >= 0 && x.Cmp(max) < 0, true)
}
