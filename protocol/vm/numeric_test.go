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
		name    string
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	tests := []struct {
		name    string
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}{
		{
			name: "test OP_1ADD",
			op:   OP_1ADD,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{0x02}},
			},
			wantVM: &virtualMachine{
				runLimit:  49998,
				dataStack: [][]byte{{0x03}},
			},
		},
		{
			name: "test OP_1SUB 2-1",
			op:   OP_1SUB,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}},
			},
			wantVM: &virtualMachine{
				runLimit:  49998,
				dataStack: [][]byte{{1}},
			},
		},
		{
			name: "test OP_1SUB use uint256's second array elem",
			op:   OP_1SUB,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -1,
				dataStack:    [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
			},
		},
		{
			name: "test OP_2MUL 2*2",
			op:   OP_2MUL,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}},
			},
			wantVM: &virtualMachine{
				runLimit:  49998,
				dataStack: [][]byte{{4}},
			},
		},
		{
			name: "test OP_2MUL use uint256's full array elem",
			op:   OP_2MUL,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x3f}},
			},
			wantVM: &virtualMachine{
				runLimit:  49998,
				dataStack: [][]byte{{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}},
			},
		},
		{
			name: "test OP_2MUL use uint256's second array elem",
			op:   OP_2MUL,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: 1,
				dataStack:    [][]byte{{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}},
			},
		},
		{
			name: "test OP_2DIV 2/2",
			op:   OP_2DIV,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}},
			},
			wantVM: &virtualMachine{
				runLimit:  49998,
				dataStack: [][]byte{{1}},
			},
		},
		{
			name: "test OP_NOT",
			op:   OP_NOT,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -1,
				dataStack:    [][]byte{{}},
			},
		},
		{
			name: "test OP_0NOTEQUAL",
			op:   OP_0NOTEQUAL,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}},
			},
			wantVM: &virtualMachine{
				runLimit:  49998,
				dataStack: [][]byte{{1}},
			},
		},
		{
			name: "test OP_ADD 2+1",
			op:   OP_ADD,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -9,
				dataStack:    [][]byte{{3}},
			},
		},
		{
			name: "test OP_SUB 2-1",
			op:   OP_SUB,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -9,
				dataStack:    [][]byte{{1}},
			},
		},
		{
			name: "test OP_MUL 2*1",
			op:   OP_MUL,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49992,
				deferredCost: -9,
				dataStack:    [][]byte{{2}},
			},
		},
		{
			name: "test OP_DIV 2/1",
			op:   OP_DIV,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49992,
				deferredCost: -9,
				dataStack:    [][]byte{{2}},
			},
		},
		{
			name: "test OP_DIV 2/0",
			op:   OP_DIV,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {}},
			},
			wantErr: ErrDivZero,
		},
		{
			name: "test OP_MOD 2%1",
			op:   OP_MOD,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49992,
				deferredCost: -10,
				dataStack:    [][]byte{{}},
			},
		},
		{
			name: "test OP_MOD 2%0",
			op:   OP_MOD,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {0}},
			},
			wantErr: ErrDivZero,
		},
		{
			name: "test OP_LSHIFT",
			op:   OP_LSHIFT,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49992,
				deferredCost: -9,
				dataStack:    [][]byte{{4}},
			},
		},
		{
			name: "test OP_RSHIFT",
			op:   OP_RSHIFT,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49992,
				deferredCost: -9,
				dataStack:    [][]byte{{1}},
			},
		},
		{
			name: "test OP_BOOLAND",
			op:   OP_BOOLAND,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -9,
				dataStack:    [][]byte{{1}},
			},
		},
		{
			name: "test OP_BOOLOR",
			op:   OP_BOOLOR,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -9,
				dataStack:    [][]byte{{1}},
			},
		},
		{
			name: "test OP_NUMEQUAL",
			op:   OP_NUMEQUAL,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -10,
				dataStack:    [][]byte{{}},
			},
		},
		{
			name: "test OP_NUMEQUALVERIFY",
			op:   OP_NUMEQUALVERIFY,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {2}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -18,
				dataStack:    [][]byte{},
			},
		},
		{
			name: "test OP_NUMEQUALVERIFY",
			op:   OP_NUMEQUALVERIFY,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{1}, {2}},
			},
			wantErr: ErrVerifyFailed,
		},
		{
			name: "test OP_NUMNOTEQUAL",
			op:   OP_NUMNOTEQUAL,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -9,
				dataStack:    [][]byte{{1}},
			},
		},
		{
			name: "test OP_LESSTHAN",
			op:   OP_LESSTHAN,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -10,
				dataStack:    [][]byte{{}},
			},
		},
		{
			name: "test OP_LESSTHANOREQUAL",
			op:   OP_LESSTHANOREQUAL,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -10,
				dataStack:    [][]byte{{}},
			},
		},
		{
			name: "test OP_GREATERTHAN",
			op:   OP_GREATERTHAN,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -9,
				dataStack:    [][]byte{{1}},
			},
		},
		{
			name: "test OP_GREATERTHANOREQUAL",
			op:   OP_GREATERTHANOREQUAL,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -9,
				dataStack:    [][]byte{{1}},
			},
		},
		{
			name: "test OP_MIN min(2,1)",
			op:   OP_MIN,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -9,
				dataStack:    [][]byte{{1}},
			},
		},
		{
			name: "test OP_MIN min(1,2)",
			op:   OP_MIN,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{1}, {2}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -9,
				dataStack:    [][]byte{{1}},
			},
		},
		{
			name: "test OP_MAX max(1,2)",
			op:   OP_MAX,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}, {1}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -9,
				dataStack:    [][]byte{{2}},
			},
		},
		{
			name: "test OP_MAX max(1,2)",
			op:   OP_MAX,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{1}, {2}},
			},
			wantVM: &virtualMachine{
				runLimit:     49998,
				deferredCost: -9,
				dataStack:    [][]byte{{2}},
			},
		},
		{
			name: "test OP_WITHIN",
			op:   OP_WITHIN,
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
		OP_1ADD, OP_1SUB, OP_2MUL, OP_2DIV, OP_NOT, OP_0NOTEQUAL,
		OP_ADD, OP_SUB, OP_MUL, OP_DIV, OP_MOD, OP_LSHIFT, OP_RSHIFT, OP_BOOLAND,
		OP_BOOLOR, OP_NUMEQUAL, OP_NUMEQUALVERIFY, OP_NUMNOTEQUAL, OP_LESSTHAN,
		OP_LESSTHANOREQUAL, OP_GREATERTHAN, OP_GREATERTHANOREQUAL, OP_MIN, OP_MAX, OP_WITHIN,
	}

	for _, op := range numops {
		tests = append(tests, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  0,
				dataStack: [][]byte{{2}, {2}, {2}},
			},
			wantErr: ErrRunLimitExceeded,
		})
	}

	for i, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			err := ops[c.op].fn(c.startVM)
			if err != c.wantErr {
				t.Errorf("case %d, op %s: got err = %v want %v", i, ops[c.op].name, err, c.wantErr)
				return
			}
			if c.wantErr != nil {
				return
			}

			if !testutil.DeepEqual(c.startVM, c.wantVM) {
				t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\n", i, ops[c.op].name, c.startVM, c.wantVM)
			}
		})
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

func TestOpMinMax(t *testing.T) {
	type args struct {
		vm *virtualMachine
		f  func(vm *virtualMachine) error
	}

	tests := []struct {
		name    string
		args    args
		want    [][]byte
		wantErr bool
	}{
		{
			name: "min of (2, 3)",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}, {0x03}},
				},
				f: opMin,
			},
			want:    [][]byte{{0x02}},
			wantErr: false,
		},

		{
			name: "max of (2, 3)",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}, {0x03}},
				},
				f: opMax,
			},
			want:    [][]byte{{0x03}},
			wantErr: false,
		},
		{
			name: "max of (two number, one number)",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, {0xff}},
				},
				f: opMax,
			},
			want:    [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
			wantErr: false,
		},
		{
			name: "min of (0, -1) got error",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{}, mocks.U256NumNegative1},
				},
				f: opMin,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "max of (-1, -1) got error",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{mocks.U256NumNegative1, mocks.U256NumNegative1},
				},
				f: opMax,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.args.f(tt.args.vm); err != nil {
				if !tt.wantErr {
					t.Errorf("opAdd() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if !testutil.DeepEqual(tt.args.vm.dataStack, tt.want) {
				t.Errorf("opAdd() error, got %v and wantErr %v", tt.args.vm.dataStack, tt.want)
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
					dataStack: [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x3f}},
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
					dataStack: [][]byte{{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40}},
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

func Test_opMul(t *testing.T) {
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
			name: "test normal mul op",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{2}, {2}},
				},
			},
			want:    [][]byte{{4}},
			wantErr: false,
		},
		{
			name: "test normal mul op of big number",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x3f}, {0x02}},
				},
			},
			want:    [][]byte{{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}},
			wantErr: false,
		},
		{
			name: "test error of mul op negative",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{mocks.U256NumNegative1, {0x02}},
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
					dataStack: [][]byte{{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40}, {0x02}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := opMul(tt.args.vm); err != nil {
				if !tt.wantErr {
					t.Errorf("opMul() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if !testutil.DeepEqual(tt.args.vm.dataStack, tt.want) {
				t.Errorf("opMul() error, got %v and wantErr %v", tt.args.vm.dataStack, tt.want)
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
			name: "Test that two number become one number after op sub",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}},
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

func Test_opSub(t *testing.T) {
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
					dataStack: [][]byte{{0x02}, {0x01}},
				},
			},
			want:    [][]byte{{0x01}},
			wantErr: false,
		},
		{
			name: "Test that two number become one number",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, {0x01}},
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
					dataStack: [][]byte{{}, {0x01}},
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
					dataStack: [][]byte{mocks.U256NumNegative1, {0x01}},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := opSub(tt.args.vm); (err != nil) != tt.wantErr {
				t.Errorf("opSub() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !testutil.DeepEqual(tt.args.vm.dataStack, tt.want) {
				t.Errorf("opSub() error, got %v and wantErr %v", tt.args.vm.dataStack, tt.want)
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
			name: "Test that two number become one number after op div",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}},
				},
			},
			want:    [][]byte{{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80}},
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
				if !tt.wantErr {
					t.Errorf("op2Div() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if !testutil.DeepEqual(tt.args.vm.dataStack, tt.want) {
				t.Errorf("op2Div() error, got %v and wantErr %v", tt.args.vm.dataStack, tt.want)
			}
		})
	}
}

func Test_opDiv(t *testing.T) {
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
					dataStack: [][]byte{{0x02}, {0x02}},
				},
			},
			want:    [][]byte{{0x01}},
			wantErr: false,
		},
		{
			name: "Test 2 div 1 = 2",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}, {0x01}},
				},
			},
			want:    [][]byte{{0x02}},
			wantErr: false,
		},
		{
			name: "Test that two number become one number after op div",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, {0x02}},
				},
			},
			want:    [][]byte{{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80}},
			wantErr: false,
		},
		{
			name: "Test for 0 div 2 got 0",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{}, {0x02}},
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
					dataStack: [][]byte{mocks.U256NumNegative1, {0x02}},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test for 1 div 0 got error",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x01}, {}},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := opDiv(tt.args.vm); err != nil {
				if !tt.wantErr {
					t.Errorf("opDiv() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if !testutil.DeepEqual(tt.args.vm.dataStack, tt.want) {
				t.Errorf("opDiv() error, got %v and wantErr %v", tt.args.vm.dataStack, tt.want)
			}
		})
	}
}

func Test_opAdd(t *testing.T) {
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
			name: "Test 2 + 2 = 4",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}, {0x02}},
				},
			},
			want:    [][]byte{{0x04}},
			wantErr: false,
		},
		{
			name: "Test that one number become two number",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, {0x01}},
				},
			},
			want:    [][]byte{{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}},
			wantErr: false,
		},
		{
			name: "Test for 0 + -1 got error",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{}, mocks.U256NumNegative1},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test for -1 + -1 got error",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{mocks.U256NumNegative1, mocks.U256NumNegative1},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := opAdd(tt.args.vm); err != nil {
				if !tt.wantErr {
					t.Errorf("opAdd() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if !testutil.DeepEqual(tt.args.vm.dataStack, tt.want) {
				t.Errorf("opAdd() error, got %v and wantErr %v", tt.args.vm.dataStack, tt.want)
			}
		})
	}
}

func Test_opMod(t *testing.T) {
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
			name: "Test 2 mod 2 = 0",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}, {0x02}},
				},
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "Test 2 mod 1 = 0",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}, {0x01}},
				},
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "Test 255 mod 4 = 3",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0xff}, {0x04}},
				},
			},
			want:    [][]byte{{0x03}},
			wantErr: false,
		},
		{
			name: "Test that two number become one number",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, {0x03}},
				},
			},
			want:    [][]byte{{0x01}},
			wantErr: false,
		},
		{
			name: "Test for 0 mod 2 got 0",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{}, {0x02}},
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
					dataStack: [][]byte{mocks.U256NumNegative1, {0x02}},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test for 1 div 0 got error",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x01}, {}},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := opMod(tt.args.vm); err != nil {
				if !tt.wantErr {
					t.Errorf("opMod() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if !testutil.DeepEqual(tt.args.vm.dataStack, tt.want) {
				t.Errorf("opMod() error, got %v and wantErr %v", tt.args.vm.dataStack, tt.want)
			}
		})
	}
}

func TestOpShift(t *testing.T) {
	type args struct {
		vm *virtualMachine
		f  func(vm *virtualMachine) error
	}

	tests := []struct {
		name    string
		args    args
		want    [][]byte
		wantErr bool
	}{
		{
			name: "2 left shift 0",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}, {}},
				},
				f: opLshift,
			},
			want:    [][]byte{{0x02}},
			wantErr: false,
		},
		{
			name: "2 right shift 0",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}, {}},
				},
				f: opRshift,
			},
			want:    [][]byte{{0x02}},
			wantErr: false,
		},
		{
			name: "2 left shift 3",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}, {0x03}},
				},
				f: opLshift,
			},
			want:    [][]byte{{0x10}},
			wantErr: false,
		},
		{
			name: "2 right shift 3",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0x02}, {0x03}},
				},
				f: opRshift,
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "two number right shift become one number",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, {0x0f}},
				},
				f: opRshift,
			},
			want:    [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}},
			wantErr: false,
		},
		{
			name: "two number left shift become overflow",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, {0xff}},
				},
				f: opLshift,
			},
			wantErr: true,
		},
		{
			name: "left shift not uint64",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0xff}, {0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
				},
				f: opLshift,
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "right shift not uint64",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{0xff}, {0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
				},
				f: opRshift,
			},
			want:    [][]byte{{}},
			wantErr: false,
		},
		{
			name: "0 left shift -1 got error",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{{}, mocks.U256NumNegative1},
				},
				f: opLshift,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "-1 right shift -1 got error",
			args: args{
				vm: &virtualMachine{
					runLimit:  50000,
					dataStack: [][]byte{mocks.U256NumNegative1, mocks.U256NumNegative1},
				},
				f: opRshift,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.args.f(tt.args.vm); err != nil {
				if !tt.wantErr {
					t.Errorf("opShift() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if !testutil.DeepEqual(tt.args.vm.dataStack, tt.want) {
				t.Errorf("opShift() error, got %v and wantErr %v", tt.args.vm.dataStack, tt.want)
			}
		})
	}
}
