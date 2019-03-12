package segwit

import (
	"testing"
	"encoding/hex"
)

func TestConvertP2SHProgram(t *testing.T) {
	cases := []struct {
		desc    string
		program string
		script  string
	}{
		{
			desc: "multi sign 2-1",
			program: "0020e402787b2bf9749f8fcdcc132a44e86bacf36780ec5df2189a11020d590533ee",
			script: "76aa20e402787b2bf9749f8fcdcc132a44e86bacf36780ec5df2189a11020d590533ee8808ffffffffffffffff7c00c0",
		},
		{
			desc: "multi sign 5-3",
			program: "0020e402787b2bf9749f8fcdcc132a44e86bacf36780ec5df2189a11020d590533ee",
			script: "76aa20e402787b2bf9749f8fcdcc132a44e86bacf36780ec5df2189a11020d590533ee8808ffffffffffffffff7c00c0",
		},
	}

	for i, c := range cases {
		progBytes, err := hex.DecodeString(c.program)
		if err != nil {
			t.Fatal(err)
		}

		gotScripts, err := ConvertP2SHProgram(progBytes)
		if c.script != hex.EncodeToString(gotScripts) {
			t.Errorf("case #%d (%s) got script:%s, expect script:%s", i, c.desc, hex.EncodeToString(gotScripts), c.script)
		}
	}
}
