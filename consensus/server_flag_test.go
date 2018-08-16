package consensus

import "testing"

func TestIsEnable(t *testing.T) {
	cases := []struct {
		baseFlag   ServiceFlag
		checkFlage ServiceFlag
		result     bool
	}{
		{
			baseFlag:   SFFullNode,
			checkFlage: SFFullNode,
			result:     true,
		},
		{
			baseFlag:   SFFullNode,
			checkFlage: SFFastSync,
			result:     false,
		},
		{
			baseFlag:   SFFullNode | SFFastSync,
			checkFlage: SFFullNode,
			result:     true,
		},
		{
			baseFlag:   SFFullNode | SFFastSync,
			checkFlage: SFFastSync,
			result:     true,
		},
	}

	for i, c := range cases {
		if c.baseFlag.IsEnable(c.checkFlage) != c.result {
			t.Errorf("test case #%d got %t, want %t", i, c.baseFlag.IsEnable(c.checkFlage), c.result)
		}
	}
}
