package vm

import "encoding/binary"

func opFalse(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	return vm.pushBool(false, false)
}

func opPushdata(vm *virtualMachine) error {
	if err := vm.applyCost(1); err != nil {
		return err
	}

	d := make([]byte, len(vm.data))
	copy(d, vm.data)
	return vm.pushDataStack(d, false)
}

func opNop(vm *virtualMachine) error {
	return vm.applyCost(1)
}

// PushDataBytes push bytes to stack
func PushDataBytes(in []byte) []byte {
	l := len(in)
	if l == 0 {
		return []byte{byte(OP_0)}
	}
	if l <= 75 {
		return append([]byte{byte(OP_DATA_1) + uint8(l) - 1}, in...)
	}
	if l < 1<<8 {
		return append([]byte{byte(OP_PUSHDATA1), uint8(l)}, in...)
	}
	if l < 1<<16 {
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], uint16(l))
		return append([]byte{byte(OP_PUSHDATA2), b[0], b[1]}, in...)
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(l))
	return append([]byte{byte(OP_PUSHDATA4), b[0], b[1], b[2], b[3]}, in...)
}

// PushDataUint64 push int64 to stack
func PushDataUint64(n uint64) []byte {
	if n == 0 {
		return []byte{byte(OP_0)}
	}

	if n >= 1 && n <= 16 {
		return []byte{uint8(OP_1) + uint8(n) - 1}
	}

	return PushDataBytes(Uint64Bytes(n))
}
