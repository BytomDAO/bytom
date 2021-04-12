// +build !linux

package strftime

import "time"

func strftime(format string, t time.Time) string {
	return StrftimePure(format, t)
}
