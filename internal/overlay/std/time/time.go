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

// String gives the text of the duration, in the format of Go.
func (d Duration) String() string {
	if d == 0 {
		return "0s"
	}
	if d < Microsecond {
		return itoa(int64(d)) + "ns"
	}
	if d < Millisecond {
		return part(int64(d), 1000) + "µs"
	}
	if d < Second {
		return part(int64(d), 1000000) + "ms"
	}
	return part(int64(d), 1000000000) + "s"
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
	for len(digits) > 0 && digits[len(digits)-1] == '0' {
		digits = digits[:len(digits)-1]
	}
	return itoa(whole) + "." + digits
}
