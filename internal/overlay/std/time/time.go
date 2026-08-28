//go:build dart

// Package time gives the clock and the sleep of Go.
package time

import "syscall/dart"

// A Duration is a count of nanoseconds.
type Duration int64

const (
	Nanosecond  Duration = 1
	Microsecond          = 1000 * Nanosecond
	Millisecond          = 1000 * Microsecond
	Second               = 1000 * Millisecond
	Minute               = 60 * Second
	Hour                 = 60 * Minute
)

// Nanoseconds gives the count of nanoseconds.
func (d Duration) Nanoseconds() int64 { return int64(d) }

// Microseconds gives the count of microseconds.
func (d Duration) Microseconds() int64 { return int64(d) / 1000 }

// Milliseconds gives the count of milliseconds.
func (d Duration) Milliseconds() int64 { return int64(d) / 1000000 }

// Seconds gives the count of seconds.
func (d Duration) Seconds() float64 { return float64(d) / 1e9 }

// String gives the text of the duration, in the format of Go. A duration of 1
// second or more holds the hour, the minute, and the second.
func (d Duration) String() string {
	if d == 0 {
		return "0s"
	}
	u := int64(d)
	negative := u < 0
	if negative {
		u = -u
	}
	out := ""
	switch {
	case u < int64(Microsecond):
		out = itoa(u) + "ns"
	case u < int64(Millisecond):
		out = part(u, 1000) + "µs"
	case u < int64(Second):
		out = part(u, 1000000) + "ms"
	default:
		out = part(u%int64(Minute), 1000000000) + "s"
		minutes := u / int64(Minute)
		if minutes > 0 {
			out = itoa(minutes%60) + "m" + out
			if hours := minutes / 60; hours > 0 {
				out = itoa(hours) + "h" + out
			}
		}
	}
	if negative {
		return "-" + out
	}
	return out
}

// Sleep gives up control for the duration.
func Sleep(d Duration) {
	dart.Sleep(int64(d))
}

// A Time is a point in time.
type Time struct {
	nanos int64
}

// Now gives the time of the clock.
func Now() Time {
	return Time{nanos: dart.NowNanos()}
}

// Unix gives the time of a count of seconds and a count of nanoseconds since
// the start of the Unix epoch.
func Unix(sec, nsec int64) Time {
	return Time{nanos: sec*1000000000 + nsec}
}

// UnixNano gives the count of nanoseconds since the start of the Unix epoch.
func (t Time) UnixNano() int64 { return t.nanos }

// Unix gives the count of seconds since the start of the Unix epoch.
func (t Time) Unix() int64 { return t.nanos / 1000000000 }

// Sub gives the duration between t and u.
func (t Time) Sub(u Time) Duration { return Duration(t.nanos - u.nanos) }

// Add gives the time after the duration.
func (t Time) Add(d Duration) Time { return Time{nanos: t.nanos + int64(d)} }

// Before reports whether t comes before u.
func (t Time) Before(u Time) bool { return t.nanos < u.nanos }

// After reports whether t comes after u.
func (t Time) After(u Time) bool { return t.nanos > u.nanos }

// IsZero reports whether t holds the zero time.
func (t Time) IsZero() bool { return t.nanos == 0 }

// String gives the text of the time, in the format of Go.
//
// The layout is `2006-01-02 15:04:05.999999999 -0700 MST`. Go writes the local
// zone, thus the offset and the name of the zone come from Dart. Go also writes
// the reading of the monotonic clock, and a Time of the overlay holds no such
// reading. `docs/parity.md` holds that gap.
func (t Time) String() string {
	offset, zone := dart.ZoneOffset(t.nanos)
	local := t.nanos + int64(offset)*1000000000
	days := floorDiv(local, 86400000000000)
	rest := local - days*86400000000000
	year, month, day := civilFromDays(days)
	out := itoa(year) + "-" + pad(int64(month), 2) + "-" + pad(int64(day), 2) +
		" " + pad(rest/3600000000000, 2) +
		":" + pad(rest/60000000000%60, 2) +
		":" + pad(rest/1000000000%60, 2)
	if nanos := rest % 1000000000; nanos != 0 {
		out += "." + trimZeros(pad(nanos, 9))
	}
	return out + " " + zoneText(offset) + " " + zone
}

// zoneText writes the offset of a zone in the format of Go. A zone west of UTC
// takes a minus sign.
func zoneText(offset int) string {
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	return sign + pad(int64(offset/3600), 2) + pad(int64(offset/60%60), 2)
}

// civilFromDays gives the year, the month, and the day of a count of days since
// the start of the Unix epoch. The algorithm holds for each proleptic Gregorian
// date.
func civilFromDays(days int64) (int64, int, int) {
	days += 719468
	era := floorDiv(days, 146097)
	dayOfEra := days - era*146097
	yearOfEra := (dayOfEra - dayOfEra/1460 + dayOfEra/36524 - dayOfEra/146096) / 365
	year := yearOfEra + era*400
	dayOfYear := dayOfEra - (365*yearOfEra + yearOfEra/4 - yearOfEra/100)
	shifted := (5*dayOfYear + 2) / 153
	day := dayOfYear - (153*shifted+2)/5 + 1
	month := shifted + 3
	if shifted >= 10 {
		month = shifted - 9
	}
	if month <= 2 {
		year++
	}
	return year, int(month), int(day)
}

// floorDiv divides and rounds toward minus infinity. The division of Go rounds
// toward zero, and a date before 1970 needs the other form.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// pad writes a value with a minimum count of digits.
func pad(v int64, width int) string {
	out := itoa(v)
	for len(out) < width {
		out = "0" + out
	}
	return out
}

// trimZeros removes each trailing zero of the digits after the point.
func trimZeros(digits string) string {
	for len(digits) > 0 && digits[len(digits)-1] == '0' {
		digits = digits[:len(digits)-1]
	}
	return digits
}

// Since gives the duration between now and t.
func Since(t Time) Duration {
	return Now().Sub(t)
}

// Until gives the duration between t and now.
func Until(t Time) Duration {
	return t.Sub(Now())
}

// Timer runs a function after a time. AfterFunc gives such a timer, and the
// overlay holds no channel form.
type Timer struct {
	handle int
}

// Stop cancels a timer. It reports false when the timer ran already.
func (t *Timer) Stop() bool {
	return dart.StopTimer(t.handle)
}

// AfterFunc runs f on a new goroutine after the duration.
func AfterFunc(d Duration, f func()) *Timer {
	return &Timer{handle: dart.StartTimer(int64(d), f)}
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	out := ""
	for v > 0 {
		out = string(rune('0'+v%10)) + out
		v /= 10
	}
	if negative {
		return "-" + out
	}
	return out
}

// part writes a value and the digits after the point, with no trailing zero.
func part(v, unit int64) string {
	whole := v / unit
	rest := v % unit
	if rest == 0 {
		return itoa(whole)
	}
	digits := ""
	for unit > 1 {
		unit /= 10
		digits += string(rune('0' + rest/unit))
		rest %= unit
	}
	return itoa(whole) + "." + trimZeros(digits)
}
