package vm

func opCheckOutput(vm *virtualMachine) error {
	err := vm.applyCost(16)
	if err != nil {
		return err
	}

	code, err := vm.pop(true)
	if err != nil {
		return err
	}
	vmVersion, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if vmVersion < 0 {
		return ErrBadValue
	}
	assetID, err := vm.pop(true)
	if err != nil {
		return err
	}
	amount, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if amount < 0 {
		return ErrBadValue
	}
	index, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if index < 0 {
		return ErrBadValue
	}

	if vm.context.CheckOutput == nil {
		return ErrContext
	}

	ok, err := vm.context.CheckOutput(uint64(index), uint64(amount), assetID, uint64(vmVersion), code, vm.expansionReserved)
	if err != nil {
		return err
	}
	return vm.pushBool(ok, true)
}

func opAsset(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.AssetID == nil {
		return ErrContext
	}
	return vm.push(*vm.context.AssetID, true)
}

func opAmount(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.Amount == nil {
		return ErrContext
	}
	return vm.pushInt64(int64(*vm.context.Amount), true)
}

func opProgram(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	return vm.push(vm.context.Code, true)
}

func opIndex(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.DestPos == nil {
		return ErrContext
	}
	return vm.pushInt64(int64(*vm.context.DestPos), true)
}

func opEntryID(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	return vm.push(vm.context.EntryID, true)
}

func opOutputID(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.SpentOutputID == nil {
		return ErrContext
	}
	return vm.push(*vm.context.SpentOutputID, true)
}

func opBlockHeight(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.BlockHeight == nil {
		return ErrContext
	}
	return vm.pushInt64(int64(*vm.context.BlockHeight), true)
}
