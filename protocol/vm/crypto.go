package vm

import (
	"crypto/sha256"
	"fmt"
	"hash"

	"golang.org/x/crypto/sha3"

	"github.com/bytom/crypto"
	"github.com/bytom/crypto/sm2"
	"github.com/bytom/crypto/sm3"
	"github.com/bytom/math/checked"
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
	err = vm.applyCost(cost)
	if err != nil {
		return err
	}
	h := hashFactory()
	_, err = h.Write(x)
	if err != nil {
		return err
	}
	return vm.push(h.Sum(nil), false)
}

func opCheckSig(vm *virtualMachine) error {
	err := vm.applyCost(1024)
	if err != nil {
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
	if len(pubkeyBytes) != sm2.PubKeySize {
		return vm.pushBool(false, true)
	}
	if len(sig) != sm2.SignatureSize {
		return vm.pushBool(false, true)
	}
	fmt.Println("=====opCheckSig=====")
	fmt.Printf("pubkeyBytes: %x\n", pubkeyBytes)
	fmt.Printf("msg: %x\n", msg)
	fmt.Printf("sig: %x\n", sig)
	fmt.Printf("result: %v\n", sm2.VerifyCompressedPubkey(sm2.PubKey(pubkeyBytes), msg, sig))
	return vm.pushBool(sm2.VerifyCompressedPubkey(sm2.PubKey(pubkeyBytes), msg, sig), true)
}

func opCheckMultiSig(vm *virtualMachine) error {
	numPubkeys, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	pubCost, ok := checked.MulInt64(numPubkeys, 1024)
	if numPubkeys < 0 || !ok {
		return ErrBadValue
	}
	err = vm.applyCost(pubCost)
	if err != nil {
		return err
	}
	numSigs, err := vm.popInt64(true)
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

	pubkeys := make([]sm2.PubKey, 0, numPubkeys)
	for _, p := range pubkeyByteses {
		if len(p) != sm2.PubKeySize {
			return vm.pushBool(false, true)
		}
		pubkeys = append(pubkeys, sm2.PubKey(p))
	}

	for len(sigs) > 0 && len(pubkeys) > 0 {
		if sm2.VerifyCompressedPubkey(pubkeys[0], msg, sigs[0]) {
			sigs = sigs[1:]
		}
		pubkeys = pubkeys[1:]
	}
	return vm.pushBool(len(sigs) == 0, true)
}

func opTxSigHash(vm *virtualMachine) error {
	err := vm.applyCost(256)
	if err != nil {
		return err
	}
	if vm.context.TxSigHash == nil {
		return ErrContext
	}
	return vm.push(vm.context.TxSigHash(), false)
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

	return vm.push(crypto.Ripemd160(data), false)
}

func opSm3(vm *virtualMachine) error {
	return doHash(vm, sm3.New)
}

func opCheckSigSm2(vm *virtualMachine) error {
	if err := vm.applyCost(1024); err != nil {
		return err
	}
	publicKey, err := vm.pop(true)
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

	if len(msg) != 32 || len(sig) != sm2.SignatureSize {
		return ErrBadValue
	}
	if len(publicKey) != sm2.PubKeySize {
		return vm.pushBool(false, true)
	}
	fmt.Println("=====opCheckSigSm2=====")
	return vm.pushBool(sm2.VerifyCompressedPubkey(publicKey, msg, sig), true)
}

func opCheckMultiSigSm2(vm *virtualMachine) error {
	numPubkeys, err := vm.popInt64(true)
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
	numSigs, err := vm.popInt64(true)
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
		if len(pubkeyBytes) != 33 {
			return vm.pushBool(false, true)
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
		if len(sig) != 64 {
			return ErrBadValue
		}
		sigs = append(sigs, sig)
	}

	for len(sigs) > 0 && len(pubkeyByteses) > 0 {
		if sm2.VerifyCompressedPubkey(pubkeyByteses[0], msg, sigs[0]) {
			sigs = sigs[1:]
		}
		pubkeyByteses = pubkeyByteses[1:]
	}
	return vm.pushBool(len(sigs) == 0, true)
}
