package bc

import (
	"reflect"
	"testing"
)

func BenchmarkEntryID(b *testing.B) {
	m := NewMux([]*ValueSource{{Position: 1}}, &Program{Code: []byte{1}, VmVersion: 1})

	entries := []Entry{
		NewIssuance(nil, &AssetAmount{}, &Hash{}, 0),
		m,
		NewTxHeader(1, 1, 0, nil),
		NewNonce(&Program{Code: []byte{1}, VmVersion: 1}),
		NewOutput(&ValueSource{}, &Program{Code: []byte{1}, VmVersion: 1}, &Hash{}, 0),
		NewRetirement(&ValueSource{}, &Hash{}, 1),
		NewSpend(&Hash{}, &Hash{}, 0),
	}

	for _, e := range entries {
		name := reflect.TypeOf(e).Elem().Name()
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				EntryID(e)
			}
		})
	}
}
