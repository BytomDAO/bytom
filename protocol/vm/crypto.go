package vm

import (
	"crypto/ed25519"
	"crypto/sha256"
	"hash"

	"golang.org/x/crypto/sha3"

	"github.com/bytom/bytom/crypto"
	"github.com/bytom/bytom/math/checked"
)

func opSha256(vm *virtualMachine) error {
	return doHash(vm, sha256.New)
}

func opSha3(vm *virtualMachine) error {
	return doHash(vm, sha3.New256)
}

func doHash(vm *virtualMachine, hashFactory func() hash.Hash) error {
	x, err := vm.pop(false)
	if err != nil {
		return err
	}

	cost := int64(len(x))
	if cost < 64 {
		cost = 64
	}

	if err = vm.applyCost(cost); err != nil {
		return err
	}

	h := hashFactory()
	if _, err = h.Write(x); err != nil {
		return err
	}
	return vm.pushDataStack(h.Sum(nil), false)
}

func opCheckSig(vm *virtualMachine) error {
	if err := vm.applyCost(1024); err != nil {
		return err
	}

	pubkeyBytes, err := vm.pop(true)
	if err != nil {
		return err
	}

	msg, err := vm.pop(true)
	if err != nil {
		return err
	}

	sig, err := vm.pop(true)
	if err != nil {
		return err
	}

	if len(msg) != 32 {
		return ErrBadValue
	}

	if len(pubkeyBytes) != ed25519.PublicKeySize {
		return vm.pushBool(false, true)
	}
	return vm.pushBool(ed25519.Verify(ed25519.PublicKey(pubkeyBytes), msg, sig), true)
}

func opCheckMultiSig(vm *virtualMachine) error {
	numPubkeysBigInt, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	numPubkeys, err := bigIntInt64(numPubkeysBigInt)
	if err != nil {
		return err
	}

	pubCost, ok := checked.MulInt64(numPubkeys, 1024)
	if numPubkeys < 0 || !ok {
		return ErrBadValue
	}

	if err = vm.applyCost(pubCost); err != nil {
		return err
	}

	numSigsBigInt, err := vm.popBigInt(true)
	if err != nil {
		return err
	}

	numSigs, err := bigIntInt64(numSigsBigInt)
	if err != nil {
		return err
	}

	if numSigs < 0 || numSigs > numPubkeys || (numPubkeys > 0 && numSigs == 0) {
		return ErrBadValue
	}

	pubkeyByteses := make([][]byte, 0, numPubkeys)
	for i := int64(0); i < numPubkeys; i++ {
		pubkeyBytes, err := vm.pop(true)
		if err != nil {
			return err
		}
		pubkeyByteses = append(pubkeyByteses, pubkeyBytes)
	}

	msg, err := vm.pop(true)
	if err != nil {
		return err
	}

	if len(msg) != 32 {
		return ErrBadValue
	}

	sigs := make([][]byte, 0, numSigs)
	for i := int64(0); i < numSigs; i++ {
		sig, err := vm.pop(true)
		if err != nil {
			return err
		}
		sigs = append(sigs, sig)
	}

	pubkeys := make([]ed25519.PublicKey, 0, numPubkeys)
	for _, p := range pubkeyByteses {
		if len(p) != ed25519.PublicKeySize {
			return vm.pushBool(false, true)
		}
		pubkeys = append(pubkeys, ed25519.PublicKey(p))
	}

	for len(sigs) > 0 && len(pubkeys) > 0 {
		if ed25519.Verify(pubkeys[0], msg, sigs[0]) {
			sigs = sigs[1:]
		}
		pubkeys = pubkeys[1:]
	}

	return vm.pushBool(len(sigs) == 0, true)
}

func opTxSigHash(vm *virtualMachine) error {
	if err := vm.applyCost(256); err != nil {
		return err
	}

	if vm.context.TxSigHash == nil {
		return ErrContext
	}

	return vm.pushDataStack(vm.context.TxSigHash(), false)
}

func opHash160(vm *virtualMachine) error {
	data, err := vm.pop(false)
	if err != nil {
		return err
	}

	cost := int64(len(data) + 64)
	if err = vm.applyCost(cost); err != nil {
		return err
	}

	return vm.pushDataStack(crypto.Ripemd160(data), false)
}
