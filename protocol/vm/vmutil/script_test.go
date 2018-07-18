package vmutil

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/bytom/protocol/vm"
)

// TestIsUnspendable ensures the IsUnspendable function returns the expected
// results.
func TestIsUnspendable(t *testing.T) {
	tests := []struct {
		pkScript []byte
		expected bool
	}{
		{
			// Unspendable
			pkScript: []byte{0x6a, 0x04, 0x74, 0x65, 0x73, 0x74},
			expected: true,
		},
		{
			// Spendable
			pkScript: []byte{0x76, 0xa9, 0x14, 0x29, 0x95, 0xa0,
				0xfe, 0x68, 0x43, 0xfa, 0x9b, 0x95, 0x45,
				0x97, 0xf0, 0xdc, 0xa7, 0xa4, 0x4d, 0xf6,
				0xfa, 0x0b, 0x5c, 0x88, 0xac},
			expected: false,
		},
	}

	for i, test := range tests {
		res := IsUnspendable(test.pkScript)
		if res != test.expected {
			t.Errorf("TestIsUnspendable #%d failed: got %v want %v",
				i, res, test.expected)
			continue
		}
	}
}

type tracebuf struct {
	bytes.Buffer
}

func (t tracebuf) dump() {
	os.Stdout.Write(t.Bytes())
}

// Programs that run without error.
func TestProgramOKSm2(t *testing.T) {
	doOKNotOKSm2(t, true)
}

// Programs that return an ErrFalseVMResult.
func TestProgramNotOKSm2(t *testing.T) {
	doOKNotOKSm2(t, false)
}

func doOKNotOKSm2(t *testing.T, expectOK bool) {
	cases := []struct {
		prog []byte
		args [][]byte
	}{
		{
			prog: tP2PKHSigSm2Program(decodeString("f0b43e94ba45accaace692ed534382eb17e6ab5a19ce7b31f4486fdfc0d28640")),
			args: [][]byte{},
		},
	}
	for i, c := range cases {
		progSrc := c.prog
		if !expectOK {
			// progSrc += " NOT"
			progSrc = append(progSrc, []byte(" NOT")...)
		}
		// prog, err := vm.Assemble(progSrc)
		prog := Builder
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("* case %d, prog [%s] [%x]\n", i, progSrc, prog)
		trace := new(tracebuf)
		vm.TraceOut = trace
		vm := &virtualMachine{
			program:   prog,
			runLimit:  int64(10000),
			dataStack: append([][]byte{}, c.args...),
		}
		err = vm.run()
		if err == nil && vm.falseResult() {
			err = ErrFalseVMResult
		}
		if expectOK && err != nil {
			trace.dump()
			t.Errorf("case %d [%s]: expected success, got error %s", i, progSrc, err)
		} else if !expectOK && err != ErrFalseVMResult {
			trace.dump()
			t.Errorf("case %d [%s]: expected ErrFalseVMResult, got %s", i, progSrc, err)
		}
	}
}

func decodeString(s string) []byte {
	data, _ := hex.DecodeString(s)

	return data
}

func tP2PKHSigSm2Program(pubkeyHash []byte) []byte {
	b, _ := P2PKHSigSm2Program(pubkeyHash)

	return b
}
