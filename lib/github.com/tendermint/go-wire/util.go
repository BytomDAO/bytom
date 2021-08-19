package wire

import (
	"bytes"

	cmn "github.com/tendermint/tmlibs/common"
)

func BinaryBytes(o interface{}) []byte {
	w, n, err := new(bytes.Buffer), new(int), new(error)
	WriteBinary(o, w, n, err)
	if *err != nil {
		cmn.PanicSanity(*err)
	}
	return w.Bytes()
}

// ptr: a pointer to the object to be filled
func ReadBinaryBytes(d []byte, ptr interface{}) error {
	r, n, err := bytes.NewBuffer(d), new(int), new(error)
	ReadBinaryPtr(ptr, r, len(d), n, err)
	return *err
}

func JSONBytes(o interface{}) []byte {
	w, n, err := new(bytes.Buffer), new(int), new(error)
	WriteJSON(o, w, n, err)
	if *err != nil {
		cmn.PanicSanity(*err)
	}
	return w.Bytes()
}
