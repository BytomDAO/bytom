package nft

import (
	"github.com/bytom/bytom/protocol/vm"
	"github.com/bytom/bytom/protocol/vm/vmutil"
)

/*
	init alt stack 	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	buy data stack 	[buyer, payAmount, marginAmount, selecter]
	edit data stack	[newMarginAsset, newMarginAmount, selecter]
*/

func NewContract(platformScript []byte, marginFold uint64) ([]byte, error) {
	builder := vmutil.NewBuilder()
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[...... selecter]
	builder.AddJumpIf(0)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, payAmount, marginAmount]
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_DUP)
	builder.AddUint64(100)
	builder.AddOp(vm.OP_DIV)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, newMarginAmount, payAmount, platformFee]
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_DUP)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, newMarginAmount, platformFee, payAmount, payAmount]
	cpAltStack(builder, 4)
	builder.AddOp(vm.OP_MUL)
	builder.AddUint64(100)
	builder.AddOp(vm.OP_DIV)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, newMarginAmount, platformFee, payAmount, createrTax]
	builder.AddOp(vm.OP_DUP)
	builder.AddUint64(2)
	builder.AddOp(vm.OP_SWAP)
	cpAltStack(builder, 1)
	builder.AddUint64(1)
	cpAltStack(builder, 5)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, newMarginAmount, platformFee, payAmount, createrTax, 2, createrTax, marginAsset, 1, PROGRAM]
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_SUB)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, newMarginAmount, platformFee, payAmount-createrTax]
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_DUP)
	builder.AddUint64(3)
	builder.AddOp(vm.OP_SWAP)
	cpAltStack(builder, 1)
	builder.AddUint64(1)
	builder.AddData(platformScript)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, newMarginAmount, payAmount-createrTax, platformFee, 3, platformFee, marginAsset, 1, platformScript]
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_SUB)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, newMarginAmount, ownerGot]
	builder.AddOp(vm.OP_DUP)
	cpAltStack(builder, 0)
	builder.AddUint64(marginFold)
	builder.AddOp(vm.OP_MUL)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, newMarginAmount, ownerGot, ownerGot, marginAmount*marginFold]
	builder.AddOp(vm.OP_GREATERTHANOREQUAL)
	builder.AddOp(vm.OP_VERIFY)
	cpAltStack(builder, 0)
	builder.AddOp(vm.OP_ADD)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, newMarginAmount, ownerGot+marginAmount]
	builder.AddUint64(4)
	builder.AddOp(vm.OP_SWAP)
	cpAltStack(builder, 1)
	builder.AddUint64(1)
	cpAltStack(builder, 2)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, newMarginAmount, 4, ownerGot+marginAmount, marginAsset, 1, owner]
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	// alt stack	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[marginAsset, buyer, newMarginAmount]
	swapAltStack(builder, 1, 2)
	swapAltStack(builder, 0, 0)
	// alt stack	[creater, taxRate, nftAsset, buyer, marginAsset, newMarginAmount]
	// data statck	[marginAsset]
	swapAltStack(builder, 0, 1)
	builder.AddJump(1)

	builder.SetJumpTarget(0)
	// alt stack 	[creater, taxRate, nftAsset, owner, marginAsset, marginAmount]
	// data statck	[newMarginAmount]
	swapAltStack(builder, 0, 0)
	// alt stack 	[creater, taxRate, nftAsset, owner, newMarginAsset, newMarginAmount]
	// data statck	[]
	builder.SetJumpTarget(1)
	builder.AddUint64(0)
	builder.AddUint64(1)
	cpAltStack(builder, 3)
	builder.AddUint64(1)
	builder.AddOp(vm.OP_PROGRAM)
	// alt stack 	[creater, taxRate, nftAsset, owner, newMarginAsset, newMarginAmount]
	// data statck	[0, 1, nftAsset, 1, PROGRAM]
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddUint64(1)
	cpAltStack(builder, 0)
	cpAltStack(builder, 1)
	builder.AddUint64(1)
	builder.AddOp(vm.OP_PROGRAM)
	// alt stack 	[creater, taxRate, nftAsset, owner, newMarginAsset, newMarginAmount]
	// data statck	[1, newMarginAmount, newMarginAsset, 1, PROGRAM]
	builder.AddOp(vm.OP_CHECKOUTPUT)
	return builder.Build()
}

func NewOffer(nftContract []byte) ([]byte, error) {
	builder := vmutil.NewBuilder()
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
