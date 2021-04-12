package wire

import (
	"io"
)

func WriteString(s string, w io.Writer, n *int, err *error) {
	WriteByteSlice([]byte(s), w, n, err)
}

func ReadString(r io.Reader, lmt int, n *int, err *error) string {
	return string(ReadByteSlice(r, lmt, n, err))
}
