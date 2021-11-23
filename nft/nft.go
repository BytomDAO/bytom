package nft

import (
	"github.com/bytom/bytom/protocol/vm"
	"github.com/bytom/bytom/protocol/vm/vmutil"
)

/*
	init alt stack 	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	buy data stack 	[newPublicKey, marginAsset, buyer, newMarginAmount, selecter]
	edit data stack	[signature, newMarginAsset, newMarginAmount, selecter]
*/

func NewContract(platformScript []byte, marginFold uint64) ([]byte, error) {
	builder := vmutil.NewBuilder()
	/*
	   For transfer nft:
	   init alt stack 	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	   init data stack	[signature, buyerPublicKey, buyerScirpt, selecter]
	*/

	// first check clause_selector for addMargin & subMargin
	builder.AddOp(vm.OP_DUP)
	builder.AddUint64(1)
	builder.AddOp(vm.OP_EQUAL)
	builder.AddUint64(2)
	builder.AddOp(vm.OP_PICK)
	cpAltStack(builder, 0)
	builder.AddOp(vm.OP_LESSTHAN)
	builder.AddOp(vm.OP_BOOLAND)
	builder.AddOp(vm.OP_NOT)
	builder.AddOp(vm.OP_VERIFY)

	builder.AddOp(vm.OP_DUP)
	builder.AddUint64(3)
	builder.AddOp(vm.OP_EQUAL)
	builder.AddJumpIf(3)
	builder.AddJump(4)

	builder.SetJumpTarget(3)
	builder.AddOp(vm.OP_DROP)

	builder.AddUint64(2)
	builder.AddOp(vm.OP_ROLL)
	cpAltStack(builder, 6)
	builder.AddOp(vm.OP_TXSIGHASH)
	builder.AddOp(vm.OP_SWAP)
	// alt stack 	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data stack	[buyerPublicKey, buyerScirpt, ownerSignature, txSigHash, publicKey]
	builder.AddOp(vm.OP_CHECKSIG)
	builder.AddOp(vm.OP_VERIFY)

	swapAltStack(builder, 0, 2)
	swapAltStack(builder, 0, 6)

	builder.AddUint64(0)
	builder.AddUint64(1)
	cpAltStack(builder, 3)
	builder.AddUint64(1)
	builder.AddOp(vm.OP_PROGRAM)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)

	builder.AddUint64(1)
	cpAltStack(builder, 0)
	cpAltStack(builder, 1)
	builder.AddUint64(1)
	builder.AddOp(vm.OP_PROGRAM)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddJump(5)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[...... selecter]
	builder.SetJumpTarget(4)
	builder.AddOp(vm.OP_DUP)
	builder.AddUint64(2)
	builder.AddOp(vm.OP_EQUAL)
	builder.AddJumpIf(2)

	builder.AddJumpIf(0)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount]
	cpAltStack(builder, 0)
	builder.AddUint64(marginFold)
	builder.AddOp(vm.OP_MUL)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount, payAmount=marginAmount*marginFold]
	builder.AddOp(vm.OP_DUP)
	builder.AddUint64(100)
	builder.AddOp(vm.OP_DIV)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount, payAmount, platformFee]
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_DUP)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount, platformFee, payAmount, payAmount]
	cpAltStack(builder, 4)
	builder.AddOp(vm.OP_MUL)
	builder.AddUint64(10000)
	builder.AddOp(vm.OP_DIV)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount, platformFee, payAmount, createrTax]
	builder.AddOp(vm.OP_DUP)
	builder.AddUint64(2)
	builder.AddOp(vm.OP_SWAP)
	cpAltStack(builder, 1)
	builder.AddUint64(1)
	cpAltStack(builder, 5)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount, platformFee, payAmount, createrTax, 2, createrTax, marginAsset, 1, creater]
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_SUB)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount, platformFee, payAmount-createrTax]
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_DUP)
	builder.AddUint64(3)
	builder.AddOp(vm.OP_SWAP)
	cpAltStack(builder, 1)
	builder.AddUint64(1)
	builder.AddData(platformScript)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount, payAmount-createrTax, platformFee, 3, platformFee, marginAsset, 1, platformScript]
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_SUB)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount, ownerGot]
	cpAltStack(builder, 0)
	builder.AddOp(vm.OP_ADD)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount, ownerGot+marginAmount]
	builder.AddUint64(4)
	builder.AddOp(vm.OP_SWAP)
	cpAltStack(builder, 1)
	builder.AddUint64(1)
	cpAltStack(builder, 2)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount, 4, ownerGot+marginAmount, marginAsset, 1, owner]
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newPublicKey, marginAsset, buyer, newMarginAmount]
	swapAltStack(builder, 1, 2)
	swapAltStack(builder, 0, 0)
	// alt stack	[publicKey, creater, taxRate, nftAsset, buyer, marginAsset, newMarginAmount]
	// data statck	[newPublicKey, marginAsset]
	swapAltStack(builder, 0, 1)
	swapAltStack(builder, 0, 6)
	// alt stack	[newPublicKey, creater, taxRate, nftAsset, buyer, marginAsset, newMarginAmount]
	// data statck	[]
	builder.AddJump(1)

	builder.SetJumpTarget(2)
	// alt stack 	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[ownerSignature, newMarginAmount]
	builder.AddOp(vm.OP_DROP)
	builder.AddOp(vm.OP_DUP)
	cpAltStack(builder, 0)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_SUB)
	builder.AddUint64(2)
	builder.AddOp(vm.OP_SWAP)
	cpAltStack(builder, 1)
	builder.AddUint64(1)
	cpAltStack(builder, 2)
	// alt stack	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[ownerSignature, newMarginAmount, 2, marginAmount-newMarginAmount, marginAsset, 1, owner]
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)

	builder.SetJumpTarget(0)
	// alt stack 	[publicKey, creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[ownerSignature, newMarginAmount]
	swapAltStack(builder, 0, 0)
	// alt stack 	[publicKey, creater, taxRate, nftAsset, owner, newMarginAsset, newMarginAmount]
	// data statck	[ownerSignature]
	cpAltStack(builder, 6)
	// alt stack 	[publicKey, creater, taxRate, nftAsset, owner, newMarginAsset, newMarginAmount]
	// data statck	[ownerSignature, publicKey]
	builder.AddOp(vm.OP_TXSIGHASH)
	builder.AddOp(vm.OP_SWAP)
	// alt stack 	[publicKey, creater, taxRate, nftAsset, owner, newMarginAsset, newMarginAmount]
	// data statck	[ownerSignature, txSigHash, publicKey]
	builder.AddOp(vm.OP_CHECKSIG)
	builder.AddOp(vm.OP_VERIFY)

	builder.SetJumpTarget(1)
	builder.AddUint64(0)
	builder.AddUint64(1)
	cpAltStack(builder, 3)
	builder.AddUint64(1)
	builder.AddOp(vm.OP_PROGRAM)
	// alt stack 	[publicKey, creater, taxRate, nftAsset, owner, newMarginAsset, newMarginAmount]
	// data statck	[0, 1, nftAsset, 1, PROGRAM]
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddUint64(1)
	cpAltStack(builder, 0)
	cpAltStack(builder, 1)
	builder.AddUint64(1)
	builder.AddOp(vm.OP_PROGRAM)
	// alt stack 	[publicKey, creater, taxRate, nftAsset, owner, newMarginAsset, newMarginAmount]
	// data statck	[1, newMarginAmount, newMarginAsset, 1, PROGRAM]
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.SetJumpTarget(5)
	return builder.Build()
}

func NewOffer(nftContract []byte) ([]byte, error) {
	builder := vmutil.NewBuilder()
	builder.AddJumpIf(0)
	// need check sig for cancel func
	cpAltStack(builder, 6)
	builder.AddOp(vm.OP_TXSIGHASH)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_CHECKSIG)
	builder.AddOp(vm.OP_VERIFY)

	builder.AddUint64(1)
	builder.AddJump(1)
	builder.SetJumpTarget(0)
	builder.AddUint64(0)
	builder.AddUint64(1)
	cpAltStack(builder, 3)
	builder.AddUint64(1)
	builder.AddData(nftContract)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddUint64(1)
	cpAltStack(builder, 0)
	cpAltStack(builder, 1)
	builder.AddUint64(1)
	builder.AddData(nftContract)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.SetJumpTarget(1)
	return builder.Build()
}

func swapAltStack(builder *vmutil.Builder, dataPos, AltPos uint64) {
	for i := uint64(0); i <= AltPos; i++ {
		builder.AddOp(vm.OP_FROMALTSTACK)
	}

	builder.AddOp(vm.OP_DROP)
	builder.AddUint64(dataPos + AltPos)
	builder.AddOp(vm.OP_ROLL)

	for i := uint64(0); i <= AltPos; i++ {
		builder.AddOp(vm.OP_TOALTSTACK)
	}
}

func cpAltStack(builder *vmutil.Builder, pos uint64) {
	for i := uint64(0); i <= pos; i++ {
		builder.AddOp(vm.OP_FROMALTSTACK)
	}

	builder.AddOp(vm.OP_DUP)

	for i := uint64(0); i <= pos; i++ {
		builder.AddOp(vm.OP_SWAP)
		builder.AddOp(vm.OP_TOALTSTACK)
	}
}
