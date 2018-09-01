package randentropy

import (
	crand "crypto/rand"
	"io"
)

var Reader io.Reader = &randEntropy{}

type randEntropy struct {
}

func (*randEntropy) Read(bytes []byte) (n int, err error) {
	readBytes := GetEntropyCSPRNG(len(bytes))
	copy(bytes, readBytes)
	return len(bytes), nil
}

func GetEntropyCSPRNG(n int) []byte {
	mainBuff := make([]byte, n)
	_, err := io.ReadFull(crand.Reader, mainBuff)
	if err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	return mainBuff
}
