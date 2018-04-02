package types

import (
	"bytes"
	"io"
	"testing"

	"github.com/bytom/protocol/bc"
)

func serialize(t *testing.T, wt io.WriterTo) []byte {
	var b bytes.Buffer
	if _, err := wt.WriteTo(&b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func mustDecodeHash(s string) (h bc.Hash) {
	if err := h.UnmarshalText([]byte(s)); err != nil {
		panic(err)
	}
	return h
}
