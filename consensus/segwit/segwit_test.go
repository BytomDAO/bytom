package segwit

import (
	"encoding/hex"
	"testing"
)

func TestConvertProgram(t *testing.T) {
	cases := []struct {
		desc    string
		program string
		script  string
		fun     func(prog []byte) ([]byte, error)
	}{
		{
			desc:    "multi sign 2-1",
			program: "0020e402787b2bf9749f8fcdcc132a44e86bacf36780ec5df2189a11020d590533ee",
			script:  "76aa20e402787b2bf9749f8fcdcc132a44e86bacf36780ec5df2189a11020d590533ee8808ffffffffffffffff7c00c0",
			fun:     ConvertP2SHProgram,
		},
		{
			desc:    "multi sign 5-3",
			program: "00200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66",
			script:  "76aa200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac668808ffffffffffffffff7c00c0",
			fun:     ConvertP2SHProgram,
		},
		{
			desc:    "single sign",
			program: "001437e1aec83a4e6587ca9609e4e5aa728db7007449",
			script:  "76ab1437e1aec83a4e6587ca9609e4e5aa728db700744988ae7cac",
			fun:     ConvertP2PKHSigProgram,
		},
	}

	for i, c := range cases {
		progBytes, err := hex.DecodeString(c.program)
		if err != nil {
			t.Fatal(err)
		}

		gotScript, err := c.fun(progBytes)
		if c.script != hex.EncodeToString(gotScript) {
			t.Errorf("case #%d (%s) got script:%s, expect script:%s", i, c.desc, hex.EncodeToString(gotScript), c.script)
		}
	}
}

func TestProgramType(t *testing.T) {
	cases := []struct {
		desc    string
		program string
		fun     func(prog []byte) bool
		yes     bool
	}{
		{
			desc:    "normal P2WPKHScript",
			program: "001437e1aec83a4e6587ca9609e4e5aa728db7007449",
			fun:     IsP2WPKHScript,
			yes:     true,
		},
		{
			desc:    "ugly P2WPKHScript",
			program: "00200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66",
			fun:     IsP2WPKHScript,
			yes:     false,
		},
		{
			desc:    "ugly P2WPKHScript",
			program: "51",
			fun:     IsP2WPKHScript,
			yes:     false,
		},
		{
			desc:    "normal P2WSHScript",
			program: "00200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66",
			fun:     IsP2WSHScript,
			yes:     true,
		},
		{
			desc:    "ugly P2WSHScript",
			program: "001437e1aec83a4e6587ca9609e4e5aa728db7007449",
			fun:     IsP2WSHScript,
			yes:     false,
		},
		{
			desc:    "ugly P2WSHScript",
			program: "51",
			fun:     IsP2WSHScript,
			yes:     false,
		},
		{
			desc:    "normal IsStraightforward",
			program: "51",
			fun:     IsStraightforward,
			yes:     true,
		},
		{
			desc:    "ugly IsStraightforward",
			program: "001437e1aec83a4e6587ca9609e4e5aa728db7007449",
			fun:     IsStraightforward,
			yes:     false,
		},
		{
			desc:    "ugly IsStraightforward",
			program: "00200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66",
			fun:     IsStraightforward,
			yes:     false,
		},
	}

	for i, c := range cases {
		progBytes, err := hex.DecodeString(c.program)
		if err != nil {
			t.Fatal(err)
		}

		if c.fun(progBytes) != c.yes {
			t.Errorf("case #%d (%s) got %t, expect %t", i, c.desc, c.fun(progBytes), c.yes)
		}
	}
}
