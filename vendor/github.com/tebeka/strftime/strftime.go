/*
Package strftime implements Python's strftime in Go

Example:
	str, err := strftime.Format("%Y/%m/%d", time.Now()) // 2019/07/07

Directives:
	%a - Locale’s abbreviated weekday name
	%A - Locale’s full weekday name
	%b - Locale’s abbreviated month name
	%B - Locale’s full month name
	%c - Locale’s appropriate date and time representation
	%d - Day of the month as a decimal number [01,31]
	%H - Hour (24-hour clock) as a decimal number [00,23]
	%I - Hour (12-hour clock) as a decimal number [01,12]
	%j - Day of year
	%m - Month as a decimal number [01,12]
	%M - Minute as a decimal number [00,59]
	%p - Locale’s equivalent of either AM or PM
	%S - Second as a decimal number [00,61]
	%U - Week number of the year
	%w - Weekday as a decimal number
	%W - Week number of the year
	%x - Locale’s appropriate date representation
	%X - Locale’s appropriate time representation
	%y - Year without century as a decimal number [00,99]
	%Y - Year with century as a decimal number
	%Z - Time zone name (no characters if no time zone exists)

Note that %c returns RFC1123 which is a bit different from what Python does
*/
package strftime

import (
	"fmt"
	"regexp"
	"time"
)

const (
	// Version is the package version
	Version = "0.1.5"

	// Week in time.Duration
	Week = time.Hour * 24 * 7
)

// See http://docs.python.org/3/library/time.html#time.strftime
var conv = map[string]string{
	"%a": "Mon",        // Locale’s abbreviated weekday name
	"%A": "Monday",     // Locale’s full weekday name
	"%b": "Jan",        // Locale’s abbreviated month name
	"%B": "January",    // Locale’s full month name
	"%c": time.RFC1123, // Locale’s appropriate date and time representation
	"%d": "02",         // Day of the month as a decimal number [01,31]
	"%H": "15",         // Hour (24-hour clock) as a decimal number [00,23]
	"%I": "03",         // Hour (12-hour clock) as a decimal number [01,12]
	"%m": "01",         // Month as a decimal number [01,12]
	"%M": "04",         // Minute as a decimal number [00,59]
	"%p": "PM",         // Locale’s equivalent of either AM or PM
	"%S": "05",         // Second as a decimal number [00,61]
	"%x": "01/02/06",   // Locale’s appropriate date representation
	"%X": "15:04:05",   // Locale’s appropriate time representation
	"%y": "06",         // Year without century as a decimal number [00,99]
	"%Y": "2006",       // Year with century as a decimal number
	"%Z": "MST",        // Time zone name (no characters if no time zone exists)
}

var (
	fmtRe = regexp.MustCompile("%[%a-zA-Z]")
)

// repl replaces % directives with right time, will panic on unknown directive
func repl(match string, t time.Time) (string, error) {
	if match == "%%" {
		return "%", nil
	}

	format, ok := conv[match]
	if ok {
		return t.Format(format), nil
	}

	switch match {
	case "%j":
		start := time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
		day := int(t.Sub(start).Hours()/24) + 1
		return fmt.Sprintf("%03d", day), nil
	case "%w":
		return fmt.Sprintf("%d", t.Weekday()), nil
	case "%W", "%U":
		start := time.Date(t.Year(), time.January, 1, 23, 0, 0, 0, time.UTC)
		week := 0
		for start.Before(t) {
			week++
			start = start.Add(Week)
		}

		return fmt.Sprintf("%02d", week), nil
	}

	return "", fmt.Errorf("unknown directive - %s", match)
}

// Format return string with % directives expanded.
// Will return error on unknown directive.
func Format(format string, t time.Time) (string, error) {
	var err error

	fn := func(match string) string {
		s, rerr := repl(match, t)
		if rerr != nil {
			err = rerr
		}
		return s
	}

	return fmtRe.ReplaceAllStringFunc(format, fn), err
}
