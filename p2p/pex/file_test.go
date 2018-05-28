package pex

import (
	"os"
	"testing"
)

func TestFileStorage(t *testing.T) {
	file := createTempFileName("TestFileStorage")
	defer os.Remove(file)

	a := NewAddrBook(file, true)
	for i := 1; i < 256; i++ {
		if err := a.addAddress(randIPv4Address(t), randIPv4Address(t)); err != nil {
			t.Fatal(err)
		}
	}

	i := 0
	for _, ka := range a.addrLookup {
		i++
		if i%7 != 0 {
			continue
		}
		if err := a.moveToOld(ka); err != nil {
			t.Fatal(err)
		}
	}

	if err := a.SaveToFile(); err != nil {
		t.Fatal(err)
	}

	// load address book b from file
	b := NewAddrBook(file, true)
	if err := b.loadFromFile(); err != nil {
		t.Fatal(err)
	}

	for key, want := range a.addrLookup {
		got, ok := b.addrLookup[key]
		if !ok {
			t.Errorf("can't find %s in loaded address book", key)
		}
		if !want.Addr.Equals(got.Addr) || !want.Src.Equals(got.Src) {
			t.Errorf("addrLookup check want %v but get %v", want, got)
		}
	}

	for i, aBucket := range a.bucketsNew {
		bBucket := b.bucketsNew[i]
		for j, want := range aBucket {
			got := bBucket[j]
			if !want.Addr.Equals(got.Addr) {
				t.Errorf("new bucket check want %v but get %v", want, got)
			}
		}
	}

	for i, aBucket := range a.bucketsOld {
		bBucket := b.bucketsOld[i]
		for j, want := range aBucket {
			got := bBucket[j]
			if !want.Addr.Equals(got.Addr) {
				t.Errorf("old bucket check want %v but get %v", want, got)
			}
		}
	}
}
