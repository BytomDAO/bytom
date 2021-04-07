package vm

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/bytom/bytom/common"
	"github.com/bytom/bytom/protocol/vm/mocks"
	"github.com/bytom/bytom/testutil"
)

func TestNumericOps(t *testing.T) {
	type testStruct struct {
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_1ADD,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0x02}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{0x03}},
		},
	}, {
		op: OP_1SUB,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{1}},
		},
	}, {
		op: OP_1SUB,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -1,
			dataStack:    [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		},
	}, {
		op: OP_2MUL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{4}},
		},
	}, {
		op: OP_2MUL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0x3f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe}},
		},
	}, {
		op: OP_2MUL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: 1,
			dataStack:    [][]byte{{0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe}},
		},
	}, {
		op: OP_2DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{1}},
		},
	}, {
		op: OP_2DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2)},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{Int64Bytes(-1)},
		},
	}, {
		op: OP_2DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-1)},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{Int64Bytes(-1)},
		},
	}, {
		op: OP_NEGATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: 7,
			dataStack:    [][]byte{Int64Bytes(-2)},
		},
	}, {
		op: OP_ABS,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{2}},
		},
	}, {
		op: OP_ABS,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2)},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -7,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_NOT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -1,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_0NOTEQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{1}},
		},
	}, {
		op: OP_ADD,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{3}},
		},
	}, {
		op: OP_SUB,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_MUL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2), {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{Int64Bytes(-2)},
		},
	}, {
		op: OP_DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2), Int64Bytes(-1)},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -23,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-3), Int64Bytes(2)},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{Int64Bytes(-1)},
		},
	}, {
		op: OP_DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {}},
		},
		wantErr: ErrDivZero,
	}, {
		op: OP_MOD,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -10,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_MOD,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-12), {10}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -16,
			dataStack:    [][]byte{{8}},
		},
	}, {
		op: OP_MOD,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {0}},
		},
		wantErr: ErrDivZero,
	}, {
		op: OP_LSHIFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{{4}},
		},
	}, {
		op: OP_LSHIFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2), {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{Int64Bytes(-4)},
		},
	}, {
		op: OP_RSHIFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_RSHIFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2), {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{Int64Bytes(-1)},
		},
	}, {
		op: OP_BOOLAND,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_BOOLOR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_NUMEQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -10,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_NUMEQUALVERIFY,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -18,
			dataStack:    [][]byte{},
		},
	}, {
		op: OP_NUMEQUALVERIFY,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}, {2}},
		},
		wantErr: ErrVerifyFailed,
	}, {
		op: OP_NUMNOTEQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_LESSTHAN,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -10,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_LESSTHANOREQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -10,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_GREATERTHAN,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_GREATERTHANOREQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_MIN,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_MIN,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}, {2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_MAX,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_MAX,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}, {2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_WITHIN,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}, {1}, {2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49996,
			deferredCost: -18,
			dataStack:    [][]byte{{1}},
		},
	}}

	numops := []Op{
		OP_1ADD, OP_1SUB, OP_2MUL, OP_2DIV, OP_NEGATE, OP_ABS, OP_NOT, OP_0NOTEQUAL,
		OP_ADD, OP_SUB, OP_MUL, OP_DIV, OP_MOD, OP_LSHIFT, OP_RSHIFT, OP_BOOLAND,
		OP_BOOLOR, OP_NUMEQUAL, OP_NUMEQUALVERIFY, OP_NUMNOTEQUAL, OP_LESSTHAN,
		OP_LESSTHANOREQUAL, OP_GREATERTHAN, OP_GREATERTHANOREQUAL, OP_MIN, OP_MAX, OP_WITHIN,
	}

	for _, op := range numops {
		cases = append(cases, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  0,
				dataStack: [][]byte{{2}, {2}, {2}},
			},
			wantErr: ErrRunLimitExceeded,
		})
	}

	for i, c := range cases {
		err := ops[c.op].fn(c.startVM)

		if err != c.wantErr {
			t.Errorf("case %d, op %s: got err = %v want %v", i, ops[c.op].name, err, c.wantErr)
			continue
		}
		if c.wantErr != nil {
			continue
		}

		if !testutil.DeepEqual(c.startVM, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\n", i, ops[c.op].name, c.startVM, c.wantVM)
		}
	}
}

func TestRangeErrs(t *testing.T) {
	cases := []struct {
		prog           string
		expectRangeErr bool
	}{
		{"0 1ADD", false},
		{fmt.Sprintf("%d 1ADD", int64(math.MinInt64)), true},
		{fmt.Sprintf("%d 1ADD", int64(math.MaxInt64)-1), false},
		{fmt.Sprintf("%s 1ADD", big.NewInt(0).SetBytes(common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")).String()), true},
		{fmt.Sprintf("%s 1ADD", big.NewInt(0).SetBytes(common.Hex2Bytes("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")).String()), true},
	}

	for i, c := range cases {
		prog, _ := Assemble(c.prog)
		vm := &virtualMachine{
			program:  prog,
			runLimit: 50000,
		}
		err := vm.run()
		switch err {
		case nil:
			if c.expectRangeErr {
				t.Errorf("case %d (%s): expected range error, got none", i, c.prog)
			}
		case ErrRange:
			if !c.expectRangeErr {
				t.Errorf("case %d (%s): got unexpected range error", i, c.prog)
			}
		default:
			if c.expectRangeErr {
				t.Errorf("case %d (%s): expected range error, got %s", i, c.prog, err)
			} else {
				t.Errorf("case %d (%s): got unexpected error %s", i, c.prog, err)
			}
		}
	}
}

func TestNumCompare(t *testing.T) {
	type args struct {
		vm *virtualMachine
		op int
	}
	tests := []struct {
		name    string
		args    args
		want    [][]byte
		wantErr bool
	}{
		{
			name: "test 2 > 1 for cmpLess",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x01}},
					runLimit:  50000,
				},
				op: cmpLess,
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "test 2 > 1 for cmpLessEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x01}},
					runLimit:  50000,
				},
				op: cmpLessEqual,
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "test 2 > 1 for cmpGreater",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x01}},
					runLimit:  50000,
				},
				op: cmpGreater,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
		{
			name: "test 2 > 1 for cmpGreaterEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x01}},
					runLimit:  50000,
				},
				op: cmpGreaterEqual,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
		{
			name: "test 2 > 1 for cmpEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x01}},
					runLimit:  50000,
				},
				op: cmpEqual,
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "test 2 > 1 for cmpNotEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x01}},
					runLimit:  50000,
				},
				op: cmpNotEqual,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
		{
			name: "test 2 == 2 for cmpLess",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x02}},
					runLimit:  50000,
				},
				op: cmpLess,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
		{
			name: "test 2 == 2 for cmpLessEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x02}},
					runLimit:  50000,
				},
				op: cmpLessEqual,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
		{
			name: "test 2 == 2 for cmpGreater",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x02}},
					runLimit:  50000,
				},
				op: cmpGreater,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
		{
			name: "test 2 == 2 for cmpGreaterEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x02}},
					runLimit:  50000,
				},
				op: cmpGreaterEqual,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
		{
			name: "test 2 == 2 for cmpEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x02}},
					runLimit:  50000,
				},
				op: cmpEqual,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
		{
			name: "test 2 == 2 for cmpNotEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x02}, {0x02}},
					runLimit:  50000,
				},
				op: cmpNotEqual,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
		{
			name: "test 1 < 2 for cmpLess",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x01}, {0x02}},
					runLimit:  50000,
				},
				op: cmpLess,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
		{
			name: "test 1 < 2 for cmpLessEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x01}, {0x02}},
					runLimit:  50000,
				},
				op: cmpLessEqual,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
		{
			name: "test 1 < 2 for cmpGreater",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x01}, {0x02}},
					runLimit:  50000,
				},
				op: cmpGreater,
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "test 1 < 2 for cmpGreaterEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x01}, {0x02}},
					runLimit:  50000,
				},
				op: cmpGreaterEqual,
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "test 1 < 2 for cmpEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x01}, {0x02}},
					runLimit:  50000,
				},
				op: cmpEqual,
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "test 1 < 2 for cmpNotEqual",
			args: args{
				vm: &virtualMachine{
					dataStack: [][]byte{{0x01}, {0x02}},
					runLimit:  50000,
				},
				op: cmpNotEqual,
			},
			want:    [][]byte{{1}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := doNumCompare(tt.args.vm, tt.args.op); (err != nil) != tt.wantErr {
				t.Errorf("doNumCompare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_op2Mul(t *testing.T) {
	type args struct {
		vm *virtualMachine
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test normal mul op",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{2}},
				},
			},
			wantErr: false,
		},
		{
			name: "test normal mul op of big number",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x3f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
				},
			},
			wantErr: false,
		},
		{
			name: "test error of mul op negative",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{mocks.U256NumNegative1},
				},
			},
			wantErr: true,
		},
		{
			name: "test error of mul op out range",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{mocks.MaxU256},
				},
			},
			wantErr: true,
		},
		{
			name: "test error of mul op out range which result is min number",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := op2Mul(tt.args.vm); (err != nil) != tt.wantErr {
				t.Errorf("op2Mul() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_op1Sub(t *testing.T) {
	type args struct {
		vm *virtualMachine
	}
	tests := []struct {
		name    string
		args    args
		want    [][]byte
		wantErr bool
	}{
		{
			name: "Test 2 - 1 = 1",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}},
				},
			},
			want:    [][]byte{{0x01}},
			wantErr: false,
		},
		{
			name: "Test that two bytes number become one byte number after op sub",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
				},
			},
			want:    [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
			wantErr: false,
		},
		{
			name: "Test for 0 - 1 got error",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{}},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test for -1 - 1 got error",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{mocks.U256NumNegative1},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := op1Sub(tt.args.vm); (err != nil) != tt.wantErr {
				t.Errorf("op1Sub() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !testutil.DeepEqual(tt.args.vm.dataStack, tt.want) {
				t.Errorf("op1Sub() error, got %v and wantErr %v", tt.args.vm.dataStack, tt.want)
			}
		})
	}
}

func Test_op2Div(t *testing.T) {
	type args struct {
		vm *virtualMachine
	}

	tests := []struct {
		name    string
		args    args
		want    [][]byte
		wantErr bool
	}{
		{
			name: "Test 2 div 2 = 1",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}},
				},
			},
			want:    [][]byte{{0x01}},
			wantErr: false,
		},
		{
			name: "Test that two bytes number become one byte number after op div",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
				},
			},
			want:    [][]byte{{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
			wantErr: false,
		},
		{
			name: "Test for 0 div 2 got 0",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{}},
				},
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "Test for -1 div 2 got error",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{mocks.U256NumNegative1},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := op2Div(tt.args.vm); err != nil {
				if !tt.wantErr{
					t.Errorf("op2Div() error = %v, wantErr %v", err, tt.wantErr)
				}else {
					return
				}
			}
			if !testutil.DeepEqual(tt.args.vm.dataStack, tt.want) {
				t.Errorf("op1Sub() error, got %v and wantErr %v", tt.args.vm.dataStack, tt.want)
			}
		})
	}
}
