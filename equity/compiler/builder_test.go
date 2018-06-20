package compiler

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/bytom/protocol/vm"
)

func TestBuilder(t *testing.T) {
	cases := []struct {
		name    string
		f       func(b *builder, stk stack)
		wantHex string
	}{
		{
			"single pushdata",
			func(b *builder, stk stack) {
				stk = b.addInt64(stk, 1)
			},
			"51",
		},
		{
			"pushdata and verify",
			func(b *builder, stk stack) {
				b.addInt64(stk, 1)
				b.addOps(stk, vm.OP_VERIFY.String(), "VERIFY")
			},
			"51",
		},
		{
			"pushdata, verify, second pushdata",
			func(b *builder, stk stack) {
				b.addInt64(stk, 1)
				b.addOps(stk, vm.OP_VERIFY.String(), "VERIFY")
				b.addInt64(stk, 2)
			},
			"516952",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stk stack
			b := &builder{}
			c.f(b, stk)
			opcodes := b.opcodes()
			opcodes = optimize(opcodes)
			got, err := vm.Assemble(opcodes)
			if err != nil {
				t.Fatal(err)
			}
			want, err := hex.DecodeString(c.wantHex)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("got %x, want %x", got, want)
			}
		})
	}
}
