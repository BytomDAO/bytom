// +build linux

package strftime

/*
#define _POSIX_SOURCE
#include <stdlib.h>
#include <time.h>
*/
import "C"

import (
	"os"
	"time"
	"unsafe"
)

const initialBufSize = 256

func strftime(format string, t time.Time) (s string) {
	if format == "" {
		return
	}

	fmt := C.CString(format)
	defer C.free(unsafe.Pointer(fmt))

	// pass timezone to strftime(3) through TZ environment var.

	// XXX: this is not threadsafe; someone may set TZ to a different value
	// between when we get and reset it. we could check that it's unchanged
	// right before setting it back, but that would leave a race between
	// testing and setting, and also a worse scenario where another thread sets
	// TZ to the same value of `zone` as this one (which can't be detected),
	// only to have us unhelpfully reset it to a now-stale value.
	//
	// since a runtime environment where different threads are stomping on TZ
	// is inherently unsafe, don't waste time trying.

	zone, _ := t.Zone()
	oldZone := os.Getenv("TZ")
	if oldZone != zone {
		defer os.Setenv("TZ", oldZone)
		os.Setenv("TZ", zone)
	}

	timep := C.time_t(t.Unix())

	var tm C.struct_tm
	C.localtime_r(&timep, &tm)

	for size := initialBufSize; ; size *= 2 {
		buf := (*C.char)(C.malloc(C.size_t(size))) // can panic
		defer C.free(unsafe.Pointer(buf))
		n := C.strftime(buf, C.size_t(size), fmt, &tm)
		if n == 0 {
			// strftime(3), unhelpfully: "Note that the return value 0 does not
			// necessarily indicate an error; for example, in many locales %p
			// yields an empty string." This leaves no definite way to
			// distinguish between the cases where the value doesn't fit and
			// where it does because the string is empty. In the latter case,
			// allocating increasingly larger buffers will never change the
			// result, so we need some heuristic for bailing out.
			//
			// Since a single 2-byte conversion sequence should not produce an
			// output longer than about 24 bytes, we conservatively allow the
			// buffer size to grow up to 20 times larger than the format string
			// before giving up.
			if size > 20*len(format) {
				return
			}
		} else if int(n) < size {
			s = C.GoStringN(buf, C.int(n))
			return
		}
	}
	return
}
