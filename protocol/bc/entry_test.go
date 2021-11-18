package bc

import (
	"reflect"
	"testing"
)

func BenchmarkEntryID(b *testing.B) {
	m := NewMux([]*ValueSource{{Position: 1}}, &Program{Code: []byte{1}, VmVersion: 1})

	entries := []Entry{
		NewIssuance(nil, &AssetAmount{}, 0),
		m,
		NewTxHeader(1, 1, 0, nil),
		NewOriginalOutput(&ValueSource{}, &Program{Code: []byte{1}, VmVersion: 1}, [][]byte{{1}}, 0),
		NewRetirement(&ValueSource{}, 1),
		NewSpend(&Hash{}, 0),
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

func TestEntryID(t *testing.T) {
	cases := []struct {
		entry         Entry
		expectEntryID string
	}{
		{
			entry:         NewIssuance(&Hash{V0: 0, V1: 1, V2: 2, V3: 3}, &AssetAmount{&AssetID{V0: 1, V1: 2, V2: 3, V3: 4}, 100}, 1),
			expectEntryID: "2d763694701dee0025d330541e213ab4f9fe652b731a93b7008737a6b869c9f5",
		},
		{
			entry: NewMux(
				[]*ValueSource{
					{
						Ref:      &Hash{V0: 0, V1: 1, V2: 2, V3: 3},
						Value:    &AssetAmount{&AssetID{V0: 1, V1: 2, V2: 3, V3: 4}, 100},
						Position: 1,
					},
				},
				&Program{VmVersion: 1, Code: []byte{1, 2, 3, 4}},
			),
			expectEntryID: "5fca548b9f57d40b852cefb0f0fe07e2945ef85a1275073230222c4b35532abf",
		},
		{
			entry: NewOriginalOutput(
				&ValueSource{
					Ref:      &Hash{V0: 4, V1: 5, V2: 6, V3: 7},
					Value:    &AssetAmount{&AssetID{V0: 1, V1: 1, V2: 1, V3: 1}, 10},
					Position: 10,
				},
				&Program{VmVersion: 1, Code: []byte{5, 5, 5, 5}},
				[][]byte{{3, 4}},
				1,
			),
			expectEntryID: "98cb887c67b5b7454092a2eaa975cb1a50a496c978b0e2ebe1f0dfe22028ed01",
		},
		{
			entry: NewRetirement(
				&ValueSource{
					Ref:      &Hash{V0: 4, V1: 5, V2: 6, V3: 7},
					Value:    &AssetAmount{&AssetID{V0: 1, V1: 1, V2: 1, V3: 1}, 10},
					Position: 10,
				},
				1,
			),
			expectEntryID: "1d3eaa538cc44105d6a640c7e72a2bfd5c9fce291a29bf424e54a6fd2c41ea89",
		},
		{
			entry:         NewSpend(&Hash{V0: 0, V1: 1, V2: 2, V3: 3}, 1),
			expectEntryID: "f2662c32ffa5f0a92919e6eca3a5829efcc1916b63139baaf4fd7d4184ea03c7",
		},
		{
			entry:         NewTxHeader(1, 100, 1000, []*Hash{&Hash{V0: 4, V1: 5, V2: 6, V3: 7}}),
			expectEntryID: "873773522863f0bd3d65feebfc7cb09f8e171208b1449e0c6df8a389b865eaa2",
		},
	}

	for _, c := range cases {
		entryID := EntryID(c.entry)
		if entryID.String() != c.expectEntryID {
			t.Errorf("the got extry id:%s is not equals to expect entry id:%s", entryID.String(), c.expectEntryID)
		}
	}
}
