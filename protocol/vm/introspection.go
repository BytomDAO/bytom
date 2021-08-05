package vm

import (
	"github.com/holiman/uint256"
)

func opCheckOutput(vm *virtualMachine) error {
	if err := vm.applyCost(16); err != nil {
		return err
	}

	code, err := vm.pop(true)
	if err != nil {
		return err
	}

	vmVersion, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	assetID, err := vm.pop(true)
	if err != nil {
		return err
	}

	amountInt, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	amount, overflow := amountInt.Uint64WithOverflow()
	if overflow {
		return ErrBadValue
	}

	index, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	if vm.context.CheckOutput == nil {
		return ErrContext
	}

	ok, err := vm.context.CheckOutput(uint64(index.Uint64()), amount, assetID, uint64(vmVersion.Uint64()), code, vm.altStack, vm.expansionReserved)
	if err != nil {
		return err
	}
	return vm.pushBool(ok, true)
}

func opAsset(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	if vm.context.AssetID == nil {
		return ErrContext
	}
	return vm.pushDataStack(*vm.context.AssetID, true)
}

func opAmount(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	if vm.context.Amount == nil {
		return ErrContext
	}

	return vm.pushBigInt(uint256.NewInt(*vm.context.Amount), true)
}

func opProgram(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	return vm.pushDataStack(vm.context.Code, true)
}

func opIndex(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	if vm.context.DestPos == nil {
		return ErrContext
	}

	return vm.pushBigInt(uint256.NewInt(*vm.context.DestPos), true)
}

func opEntryID(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}
	return vm.pushDataStack(vm.context.EntryID, true)
}

func opOutputID(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	if vm.context.SpentOutputID == nil {
		return ErrContext
	}
	return vm.pushDataStack(*vm.context.SpentOutputID, true)
}

func opBlockHeight(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	if vm.context.BlockHeight == nil {
		return ErrContext
	}

	return vm.pushBigInt(uint256.NewInt(*vm.context.BlockHeight), true)
}
