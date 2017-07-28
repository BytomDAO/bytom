package wire

import (
	"io"
	"time"

	cmn "github.com/tendermint/tmlibs/common"
)

/*
Writes nanoseconds since epoch but with millisecond precision.
This is to ease compatibility with Javascript etc.
*/

func WriteTime(t time.Time, w io.Writer, n *int, err *error) {
	nanosecs := t.UnixNano()
	millisecs := nanosecs / 1000000
	WriteInt64(millisecs*1000000, w, n, err)
}

func ReadTime(r io.Reader, n *int, err *error) time.Time {
	t := ReadInt64(r, n, err)
	if t%1000000 != 0 {
		cmn.PanicSanity("Time cannot have sub-millisecond precision")
	}
	return time.Unix(0, t)
}
