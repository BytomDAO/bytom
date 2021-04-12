package strftime

import (
	"strconv"
	"time"
)

// StrftimePure is a locale-unaware implementation of strftime(3). It does not
// correctly account for locale-specific conversion specifications, so formats
// like `%c` may behave differently from the underlying platform. Additionally,
// the `%E` and `%O` modifiers are passed through as raw strings.
//
// The implementation of locale-specific conversions attempts to mirror the
// strftime(3) implementation in glibc 2.15 under LC_TIME=C.
func StrftimePure(format string, t time.Time) string {
	buf := make([]byte, 0, 512)
	for i := 0; i < len(format); {
		c := format[i]
		if c != '%' {
			buf = append(buf, c)
			i++
			continue
		}
		i++
		if i == len(format) {
			buf = append(buf, '%')
			break
		}
		b := format[i]

		switch b {
		default:
			buf = append(buf, '%', b)
		case 'a':
			// The abbreviated weekday name according to the current locale.
			buf = append(buf, t.Format("Mon")...)
		case 'A':
			// The full weekday name according to the current locale.
			buf = append(buf, t.Format("Monday")...)
		case 'b':
			// The abbreviated month name according to the current locale.
			buf = append(buf, t.Format("Jan")...)
		case 'B':
			// The full month name according to the current locale.
			buf = append(buf, t.Month().String()...)
		case 'C':
			// The century number (year/100) as a 2-digit integer. (SU)
			buf = zero2d(buf, int(t.Year())/100)
		case 'c':
			// The preferred date and time representation for the current locale.
			buf = append(buf, t.Format("Mon Jan  2 15:04:05 2006")...)
		case 'd':
			// The day of the month as a decimal number (range 01 to 31).
			buf = zero2d(buf, t.Day())
		case 'D':
			// Equivalent to %m/%d/%y. (Yecch—for Americans only. Americans should note that in other countries %d/%m/%y is rather com‐ mon. This means that in international context this format is ambiguous and should not be used.) (SU)
			buf = zero2d(buf, int(t.Month()))
			buf = append(buf, '/')
			buf = zero2d(buf, t.Day())
			buf = append(buf, '/')
			buf = zero2d(buf, t.Year()%100)
		case 'E':
			// Modifier: use alternative format, see below. (SU)
			if i+1 < len(format) {
				i++
				buf = append(buf, '%', 'E', format[i])

			} else {
				buf = append(buf, "%E"...)
			}
		case 'e':
			// Like %d, the day of the month as a decimal number, but a leading zero is replaced by a space. (SU)
			buf = twoD(buf, t.Day())
		case 'F':
			// Equivalent to %Y-%m-%d (the ISO 8601 date format). (C99)
			buf = zero4d(buf, t.Year())
			buf = append(buf, '-')
			buf = zero2d(buf, int(t.Month()))
			buf = append(buf, '-')
			buf = zero2d(buf, t.Day())
		case 'G':
			// The ISO 8601 week-based year (see NOTES) with century as a decimal number. The 4-digit year corresponding to the ISO week number (see %V). This has the same format and value as %Y, except that if the ISO week number belongs to the previous or next year, that year is used instead. (TZ)
			year, _ := t.ISOWeek()
			buf = zero4d(buf, year)
		case 'g':
			// Like %G, but without century, that is, with a 2-digit year (00-99). (TZ)
			year, _ := t.ISOWeek()
			buf = zero2d(buf, year%100)
		case 'h':
			// Equivalent to %b. (SU)
			buf = append(buf, t.Format("Jan")...)
		case 'H':
			// The hour as a decimal number using a 24-hour clock (range 00 to 23).
			buf = zero2d(buf, t.Hour())
		case 'I':
			// The hour as a decimal number using a 12-hour clock (range 01 to 12).
			buf = zero2d(buf, t.Hour()%12)
		case 'j':
			// The day of the year as a decimal number (range 001 to 366).
			buf = zero3d(buf, t.YearDay())
		case 'k':
			// The hour (24-hour clock) as a decimal number (range 0 to 23); single digits are preceded by a blank. (See also %H.) (TZ)
			buf = twoD(buf, t.Hour())
		case 'l':
			// The hour (12-hour clock) as a decimal number (range 1 to 12); single digits are preceded by a blank. (See also %I.) (TZ)
			buf = twoD(buf, t.Hour()%12)
		case 'm':
			// The month as a decimal number (range 01 to 12).
			buf = zero2d(buf, int(t.Month()))
		case 'M':
			// The minute as a decimal number (range 00 to 59).
			buf = zero2d(buf, t.Minute())
		case 'n':
			// A newline character. (SU)
			buf = append(buf, '\n')
		case 'O':
			// Modifier: use alternative format, see below. (SU)
			if i+1 < len(format) {
				i++
				buf = append(buf, '%', 'O', format[i])
			} else {
				buf = append(buf, "%O"...)
			}
		case 'p':
			// Either "AM" or "PM" according to the given time value, or the corresponding strings for the current locale. Noon is treated as "PM" and midnight as "AM".
			buf = appendAMPM(buf, t.Hour())
		case 'P':
			// Like %p but in lowercase: "am" or "pm" or a corresponding string for the current locale. (GNU)
			buf = appendampm(buf, t.Hour())
		case 'r':
			// The time in a.m. or p.m. notation. In the POSIX locale this is equivalent to %I:%M:%S %p. (SU)
			h := t.Hour()
			buf = zero2d(buf, h%12)
			buf = append(buf, ':')
			buf = zero2d(buf, t.Minute())
			buf = append(buf, ':')
			buf = zero2d(buf, t.Second())
			buf = append(buf, ' ')
			buf = appendAMPM(buf, h)
		case 'R':
			// The time in 24-hour notation (%H:%M). (SU) For a version including the seconds, see %T below.
			buf = zero2d(buf, t.Hour())
			buf = append(buf, ':')
			buf = zero2d(buf, t.Minute())
		case 's':
			// The number of seconds since the Epoch, 1970-01-01 00:00:00 +0000 (UTC). (TZ)
			buf = strconv.AppendInt(buf, t.Unix(), 10)
		case 'S':
			// The second as a decimal number (range 00 to 60). (The range is up to 60 to allow for occasional leap seconds.)
			buf = zero2d(buf, t.Second())
		case 't':
			// A tab character. (SU)
			buf = append(buf, '\t')
		case 'T':
			// The time in 24-hour notation (%H:%M:%S). (SU)
			buf = zero2d(buf, t.Hour())
			buf = append(buf, ':')
			buf = zero2d(buf, t.Minute())
			buf = append(buf, ':')
			buf = zero2d(buf, t.Second())
		case 'u':
			// The day of the week as a decimal, range 1 to 7, Monday being 1. See also %w. (SU)
			day := byte(t.Weekday())
			if day == 0 {
				day = 7
			}
			buf = append(buf, '0'+day)
		case 'U':
			// The week number of the current year as a decimal number, range 00 to 53, starting with the first Sunday as the first day of week 01. See also %V and %W.
			buf = zero2d(buf, (t.YearDay()-int(t.Weekday())+7)/7)
		case 'V':
			// The ISO 8601 week number (see NOTES) of the current year as a decimal number, range 01 to 53, where week 1 is the first week that has at least 4 days in the new year. See also %U and %W. (SU)
			_, week := t.ISOWeek()
			buf = zero2d(buf, week)
		case 'w':
			// The day of the week as a decimal, range 0 to 6, Sunday being 0. See also %u.
			buf = strconv.AppendInt(buf, int64(t.Weekday()), 10)
		case 'W':
			// The week number of the current year as a decimal number, range 00 to 53, starting with the first Monday as the first day of week 01.
			buf = zero2d(buf, (t.YearDay()-(int(t.Weekday())-1+7)%7+7)/7)
		case 'x':
			// The preferred date representation for the current locale without the time.
			buf = zero2d(buf, int(t.Month()))
			buf = append(buf, '/')
			buf = zero2d(buf, t.Day())
			buf = append(buf, '/')
			buf = zero2d(buf, t.Year()%100)
		case 'X':
			// The preferred time representation for the current locale without the date.
			buf = zero2d(buf, t.Hour())
			buf = append(buf, ':')
			buf = zero2d(buf, t.Minute())
			buf = append(buf, ':')
			buf = zero2d(buf, t.Second())
		case 'y':
			// The year as a decimal number without a century (range 00 to 99).
			buf = zero2d(buf, t.Year()%100)
		case 'Y':
			// The year as a decimal number including the century.
			buf = zero4d(buf, t.Year())
		case 'z':
			// The +hhmm or -hhmm numeric timezone (that is, the hour and minute offset from UTC). (SU)
			buf = append(buf, t.Format("-0700")...)
		case 'Z':
			// The timezone or name or abbreviation.
			buf = append(buf, t.Format("MST")...)
		case '+':
			// The date and time in date(1) format. (TZ) (Not supported in glibc2.)
			buf = append(buf, t.Format("Mon Jan _2 15:04:05 MST 2006")...)
		case '%':
			// A literal '%' character.
			buf = append(buf, '%')
		}
		i++
	}
	return string(buf)
}

// helper function to append %2d ints
func twoD(p []byte, n int) []byte {
	if n < 10 {
		return append(p, ' ', '0'+byte(n))
	}
	return strconv.AppendInt(p, int64(n), 10)
}

// helper function to append %02d ints
func zero2d(p []byte, n int) []byte {
	if n < 10 {
		return append(p, '0', '0'+byte(n))
	}

	return strconv.AppendInt(p, int64(n), 10)
}

// helper function to append %03d ints
func zero3d(p []byte, n int) []byte {
	switch {
	case n < 10:
		p = append(p, "00"...)
	case n < 100:
		p = append(p, '0')
	}
	return strconv.AppendInt(p, int64(n), 10)
}

// helper function to append %04d ints
func zero4d(p []byte, n int) []byte {
	switch {
	case n < 10:
		p = append(p, "000"...)
	case n < 100:
		p = append(p, "00"...)
	case n < 1000:
		p = append(p, '0')
	}
	return strconv.AppendInt(p, int64(n), 10)
}

func appendampm(p []byte, h int) []byte {
	var m string
	if h < 12 {
		m = "am"
	} else {
		m = "pm"
	}
	return append(p, m...)
}
func appendAMPM(p []byte, h int) []byte {
	var m string
	if h < 12 {
		m = "AM"
	} else {
		m = "PM"
	}
	return append(p, m...)
}
