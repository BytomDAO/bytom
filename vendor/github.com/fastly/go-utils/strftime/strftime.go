// Package strftime provides the POSIX strftime(3) function.
package strftime

import "time"

/*
Strftime implements the POSIX strftime(3) function. The underlying
implementation uses the native C library function on Linux and Darwin, and a
pure Go replacement otherwise. There are some functional differences between
the implementations; see also the documentation of StrftimePure.

The following documentation is adapted from the linux man-pages project.

The Strftime function formats the time `t` according to the format
specification `format` and places the result in the string s.

The format specification is a string and may contain special character
sequences called conversion specifications, each of which is introduced by a
'%' character and terminated by some other character known as a conversion
specifier character. All other character sequences are ordinary character
sequences.

The characters of ordinary character sequences are copied verbatim from format
to s. However, the characters of conversion specifications are replaced as
follows:

	%a The abbreviated weekday name according to the current locale.
	%A The full weekday name according to the current locale.
	%b The abbreviated month name according to the current locale.
	%B The full month name according to the current locale.
	%c The preferred date and time representation for the current locale.
	%C The century number (year/100) as a 2-digit integer. (SU)
	%d The day of the month as a decimal number (range 01 to 31).
	%D Equivalent to %m/%d/%y. (Yecch—for Americans only. Americans should note that in other countries %d/%m/%y is rather common. This means that in international context this format is ambiguous and should not be used.) (SU)
	%e Like %d, the day of the month as a decimal number, but a leading zero is replaced by a space. (SU)
	%E Modifier: use alternative format, see below. (SU)
	%F Equivalent to %Y-%m-%d (the ISO 8601 date format). (C99)
	%G The ISO 8601 week-based year (see NOTES) with century as a decimal number. The 4-digit year corresponding to the ISO week number (see %V). This has the same format and value as %Y, except that if the ISO week number belongs to the previous or next year, that year is used instead. (TZ)
	%g Like %G, but without century, that is, with a 2-digit year (00-99). (TZ)
	%h Equivalent to %b. (SU)
	%H The hour as a decimal number using a 24-hour clock (range 00 to 23).
	%I The hour as a decimal number using a 12-hour clock (range 01 to 12).
	%j The day of the year as a decimal number (range 001 to 366).
	%k The hour (24-hour clock) as a decimal number (range 0 to 23); single digits are preceded by a blank. (See also %H.) (TZ)
	%l The hour (12-hour clock) as a decimal number (range 1 to 12); single digits are preceded by a blank. (See also %I.) (TZ)
	%m The month as a decimal number (range 01 to 12).
	%M The minute as a decimal number (range 00 to 59).
	%n A newline character. (SU)
	%O Modifier: use alternative format, see below. (SU)
	%p Either "AM" or "PM" according to the given time value, or the corresponding strings for the current locale. Noon is treated as "PM" and midnight as "AM".
	%P Like %p but in lowercase: "am" or "pm" or a corresponding string for the current locale. (GNU)
	%r The time in a.m. or p.m. notation. In the POSIX locale this is equivalent to %I:%M:%S %p. (SU)
	%R The time in 24-hour notation (%H:%M). (SU) For a version including the seconds, see %T below.
	%s The number of seconds since the Epoch, 1970-01-01 00:00:00 +0000 (UTC). (TZ)
	%S The second as a decimal number (range 00 to 60). (The range is up to 60 to allow for occasional leap seconds.)
	%t A tab character. (SU)
	%T The time in 24-hour notation (%H:%M:%S). (SU)
	%u The day of the week as a decimal, range 1 to 7, Monday being 1. See also %w. (SU)
	%U The week number of the current year as a decimal number, range 00 to 53, starting with the first Sunday as the first day of week 01. See also %V and %W.
	%V The ISO 8601 week number (see NOTES) of the current year as a decimal number, range 01 to 53, where week 1 is the first week that has at least 4 days in the new year. See also %U and %W. (SU)
	%w The day of the week as a decimal, range 0 to 6, Sunday being 0. See also %u.
	%W The week number of the current year as a decimal number, range 00 to 53, starting with the first Monday as the first day of week 01.
	%x The preferred date representation for the current locale without the time.
	%X The preferred time representation for the current locale without the date.
	%y The year as a decimal number without a century (range 00 to 99).
	%Y The year as a decimal number including the century.
	%z The +hhmm or -hhmm numeric timezone (that is, the hour and minute offset from UTC). (SU)
	%Z The timezone or name or abbreviation.
	%+ The date and time in date(1) format. (TZ) (Not supported in glibc2.)
	%% A literal '%' character.

Some conversion specifications can be modified by preceding the conversion
specifier character by the E or O modifier to indicate that an alternative
format should be used. If the alternative format or specification does not
exist for the current locale, the behavior will be as if the unmodified
conversion specification were used. (SU) The Single UNIX Specification mentions
%Ec, %EC, %Ex, %EX, %Ey, %EY, %Od, %Oe, %OH, %OI, %Om, %OM, %OS, %Ou, %OU, %OV,
%Ow, %OW, %Oy, where the effect of the O modifier is to use alternative numeric
symbols (say, roman numerals), and that of the E modifier is to use a
locale-dependent alternative representation.

*/
func Strftime(format string, t time.Time) (s string) {
	return strftime(format, t)
}
