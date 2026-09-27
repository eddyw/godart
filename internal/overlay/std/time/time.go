// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
//
// The magnitude reads a uint64. The smallest Duration is -2 to the power 63,
// and its magnitude is 2 to the power 63, which does not fit an int64.
func (d Duration) String() string {
	if d == 0 {
		return "0s"
	}
	negative := d < 0
	u := uint64(d)
	if negative {
		u = -u
	}
	out := ""
	switch {
	case u < uint64(Microsecond):
		out = uitoa(u) + "ns"
	case u < uint64(Millisecond):
		out = upart(u, 1000) + "µs"
	case u < uint64(Second):
		out = upart(u, 1000000) + "ms"
	default:
		out = upart(u%uint64(Minute), 1000000000) + "s"
		minutes := u / uint64(Minute)
		if minutes > 0 {
			out = uitoa(minutes%60) + "m" + out
			if hours := minutes / 60; hours > 0 {
				out = uitoa(hours) + "h" + out
			}
		}
	}
	if negative {
		return "-" + out
	}
	return out
}

// uitoa writes an unsigned value.
func uitoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	out := ""
	for v > 0 {
		out = string(rune('0'+v%10)) + out
		v /= 10
	}
	return out
}

// upart writes an unsigned value and the digits after the point, with no
// trailing zero.
func upart(v, unit uint64) string {
	whole := v / unit
	rest := v % unit
	if rest == 0 {
		return uitoa(whole)
	}
	digits := ""
	for unit > 1 {
		unit /= 10
		digits += string(rune('0' + rest/unit))
		rest %= unit
	}
	return uitoa(whole) + "." + trimZeros(digits)
}

// Sleep gives up control for the duration.
func Sleep(d Duration) {
	dart.Sleep(int64(d))
}

// A Time is a point in time.
//
// It holds a count of seconds since January 1 of the year 1. It also holds a
// count of nanoseconds inside that second, and the location of its wall clock.
//
// Go holds the same origin. The zero value of the 2 runtimes is thus midnight
// UTC of January 1 of the year 1. Values from Now also hold a monotonic clock.
type Time struct {
	// wall holds nanoseconds in bits 0–29. Bit 63 selects monotonic time.
	// With that bit set, bits 30–62 hold seconds after 1885, and ext holds
	// monotonic nanoseconds. Otherwise, ext holds seconds since year 1.
	wall uint64
	ext  int64
	loc  *Location
}

const wallToInternal int64 = (1884*365 + 1884/4 - 1884/100 + 1884/400) * 86400
const (
	hasMonotonic = 1 << 63
	minWall      = wallToInternal
	maxWall      = wallToInternal + (1 << 33) - 1
	nsecMask     = 1<<30 - 1
	nsecShift    = 30
)

func (t *Time) nsec() int32 {
	return int32(t.wall & nsecMask)
}

func (t *Time) sec() int64 {
	if t.wall&hasMonotonic != 0 {
		return wallToInternal + int64(t.wall<<1>>(nsecShift+1))
	}
	return t.ext
}

func (t *Time) addSec(d int64) {
	if t.wall&hasMonotonic != 0 {
		sec := int64(t.wall << 1 >> (nsecShift + 1))
		dsec := sec + d
		if 0 <= dsec && dsec <= 1<<33-1 {
			t.wall = t.wall&nsecMask | uint64(dsec)<<nsecShift | hasMonotonic
			return
		}
		// Wall second now out of range for packed field.
		// Move to ext.
		t.stripMono()
	}

	// Check if the sum of t.ext and d overflows and handle it properly.
	sum := t.ext + d
	if (sum > t.ext) == (d > 0) {
		t.ext = sum
	} else if d > 0 {
		t.ext = 1<<63 - 1
	} else {
		t.ext = -(1<<63 - 1)
	}
}

func (t *Time) setLoc(loc *Location) {
	if loc == &utcLoc {
		loc = nil
	}
	t.stripMono()
	t.loc = loc
}

func (t *Time) stripMono() {
	if t.wall&hasMonotonic != 0 {
		t.ext = t.sec()
		t.wall &= nsecMask
	}
}

func (t *Time) setMono(m int64) {
	if t.wall&hasMonotonic == 0 {
		sec := t.ext
		if sec < minWall || maxWall < sec {
			return
		}
		t.wall |= hasMonotonic | uint64(sec-minWall)<<nsecShift
	}
	t.ext = m
}

func (t *Time) mono() int64 {
	if t.wall&hasMonotonic == 0 {
		return 0
	}
	return t.ext
}

// unixSec gives the count of seconds of t since the start of the Unix epoch.
func (t Time) unixSec() int64 { return t.sec() - unixToInternal }

// unixTime builds a Time of 2 counts. The first 1 holds seconds since the
// start of the Unix epoch, and the second 1 nanoseconds inside that second.
func unixTime(sec int64, nsec int32) Time {
	return Time{ext: sec + unixToInternal, wall: uint64(nsec), loc: Local}
}

// Now gives the time of the clock.
func Now() Time {
	nanos := dart.NowNanos()
	t := unixTime(floorDiv(nanos, 1000000000),
		int32(floorMod(nanos, 1000000000)))
	t.setMono(dart.MonoNanos() + 1)
	return t
}

// Unix gives the time of a count of seconds and a count of nanoseconds since
// the start of the Unix epoch. A count of nanoseconds outside the range 0 to
// 999999999 moves into the count of seconds, exactly as in Go.
func Unix(sec, nsec int64) Time {
	if nsec < 0 || nsec >= 1000000000 {
		n := floorDiv(nsec, 1000000000)
		sec += n
		nsec -= n * 1000000000
	}
	return unixTime(sec, int32(nsec))
}

// UnixNano gives the count of nanoseconds since the start of the Unix epoch.
//
// The result is undefined outside the year range 1678 to 2262. The count does
// not fit an int64 there, and Go names the same bound.
func (t Time) UnixNano() int64 {
	return t.unixSec()*1000000000 + int64(t.nsec())
}

// Unix gives the count of seconds since the start of the Unix epoch.
func (t Time) Unix() int64 { return t.unixSec() }

// Sub gives the duration between t and u. A result outside the range of a
// Duration gives the largest or the smallest Duration, exactly as in Go.
func (t Time) Sub(u Time) Duration {
	if t.wall&u.wall&hasMonotonic != 0 {
		d := Duration(t.ext - u.ext)
		if t.ext < u.ext && d > 0 {
			return minDuration
		}
		if t.ext > u.ext && d < 0 {
			return maxDuration
		}
		return d
	}
	t.stripMono()
	u.stripMono()
	d := Duration(t.sec()-u.sec())*Second + Duration(t.nsec()-u.nsec())
	switch {
	case u.Add(d).Equal(t):
		return d
	case t.Before(u):
		return minDuration
	default:
		return maxDuration
	}
}

// Add gives the time after the duration.
func (t Time) Add(d Duration) Time {
	dsec := int64(d / 1e9)
	nsec := t.nsec() + int32(d%1e9)
	if nsec >= 1e9 {
		dsec++
		nsec -= 1e9
	} else if nsec < 0 {
		dsec--
		nsec += 1e9
	}
	t.wall = t.wall&^nsecMask | uint64(nsec) // update nsec
	t.addSec(dsec)
	if t.wall&hasMonotonic != 0 {
		te := t.ext + int64(d)
		if d < 0 && te > t.ext || d > 0 && te < t.ext {
			// Monotonic clock reading now out of range; degrade to wall-only.
			t.stripMono()
		} else {
			t.ext = te
		}
	}
	return t
}

// Before reports whether t comes before u.
func (t Time) Before(u Time) bool {
	if t.wall&u.wall&hasMonotonic != 0 {
		return t.ext < u.ext
	}
	return t.sec() < u.sec() || t.sec() == u.sec() && t.nsec() < u.nsec()
}

// After reports whether t comes after u.
func (t Time) After(u Time) bool {
	if t.wall&u.wall&hasMonotonic != 0 {
		return t.ext > u.ext
	}
	return t.sec() > u.sec() || t.sec() == u.sec() && t.nsec() > u.nsec()
}

// Compare gives -1 when t comes before u, 0 when the 2 name the same instant,
// and 1 when t comes after u.
func (t Time) Compare(u Time) int {
	if t.wall&u.wall&hasMonotonic != 0 {
		switch {
		case t.ext < u.ext:
			return -1
		case t.ext > u.ext:
			return 1
		default:
			return 0
		}
	}
	switch {
	case t.sec() < u.sec():
		return -1
	case t.sec() > u.sec():
		return 1
	case t.nsec() < u.nsec():
		return -1
	case t.nsec() > u.nsec():
		return 1
	}
	return 0
}

// IsZero reports whether t holds the zero time, which is midnight UTC of
// January 1 of the year 1.
func (t Time) IsZero() bool { return t.sec() == 0 && t.nsec() == 0 }

// String gives the text of the time, in the format of Go.
func (t Time) String() string {
	out := t.Format("2006-01-02 15:04:05.999999999 -0700 MST")
	if t.wall&hasMonotonic == 0 {
		return out
	}
	sign := "+"
	u := uint64(t.ext)
	if t.ext < 0 {
		sign = "-"
		u = -u
	}
	frac := uitoa(u % 1000000000)
	for len(frac) < 9 {
		frac = "0" + frac
	}
	return out + " m=" + sign + uitoa(u/1000000000) + "." + frac
}

// Location gives the location of the wall clock of t. A Time with no location
// reads UTC.
func (t Time) Location() *Location {
	if t.loc == nil {
		return UTC
	}
	return t.loc
}

// In gives the time with the wall clock of the location. The instant stays
// equal.
func (t Time) In(loc *Location) Time {
	if loc == nil {
		panic("time: missing Location in call to Time.In")
	}
	t.setLoc(loc)
	return t
}

// UTC gives t with the location UTC.
func (t Time) UTC() Time { return t.In(&utcLoc) }

// Local gives t with the location of the system.
func (t Time) Local() Time { return t.In(Local) }

// Equal reports whether t and u name the same instant. The location of each
// one changes nothing.
func (t Time) Equal(u Time) bool {
	if t.wall&u.wall&hasMonotonic != 0 {
		return t.ext == u.ext
	}
	return t.sec() == u.sec() && t.nsec() == u.nsec()
}

// locWall gives the name of the zone, the offset in seconds east of UTC, and
// the reading of the wall clock of the location in seconds since the start of
// the Unix epoch.
func (t Time) locWall() (string, int, int64) {
	offset, name := t.Location().lookup(t.unixSec())
	return name, offset, t.unixSec() + int64(offset)
}

// nanosecond gives the count of nanoseconds inside the second of t. A zone
// offset holds a whole count of seconds, thus the location changes nothing.
func (t Time) nanosecond() int { return int(t.nsec()) }

// Date gives the Time of a date and a clock in a location. A value outside its
// range moves into the next unit, exactly as in Go.
func Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location) Time {
	if loc == nil {
		panic("time: missing Location in call to Date")
	}
	sec, nsec = norm(sec, nsec, 1000000000)
	min, sec = norm(min, sec, 60)
	hour, min = norm(hour, min, 60)
	day, hour = norm(day, hour, 24)
	m := int(month) - 1
	year, m = norm(year, m, 12)
	month = Month(m) + 1

	wall := daysFromCivil(int64(year), int(month), day)*86400 +
		int64(hour)*3600 + int64(min)*60 + int64(sec)
	return Time{ext: wall - int64(loc.offsetOf(wall)) + unixToInternal,
		wall: uint64(nsec)}.In(loc)
}

// norm moves a value that stands outside its base into the next unit.
func norm(hi, lo, base int) (int, int) {
	if lo < 0 {
		n := (-lo-1)/base + 1
		hi -= n
		lo += n * base
	}
	if lo >= base {
		n := lo / base
		hi += n
		lo -= n * base
	}
	return hi, lo
}

// A Month names a month of the year.
type Month int

const (
	January Month = 1 + iota
	February
	March
	April
	May
	June
	July
	August
	September
	October
	November
	December
)

var longMonthNames = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

// String gives the name of the month in English.
func (m Month) String() string {
	if January <= m && m <= December {
		return longMonthNames[int(m)-1]
	}
	return "%!Month(" + itoa(int64(m)) + ")"
}

// A Weekday names a day of the week.
type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

// shortMonthNames holds the name of each month of 3 letters. Parse reads it.
var shortMonthNames = []string{
	"Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
}

var longDayNames = []string{
	"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday",
}

// String gives the name of the day in English.
func (d Weekday) String() string {
	if Sunday <= d && d <= Saturday {
		return longDayNames[int(d)]
	}
	return "%!Weekday(" + itoa(int64(d)) + ")"
}

// Year gives the year of t in its location.
func (t Time) Year() int { year, _, _ := t.Date(); return year }

// Month gives the month of the year of t in its location.
func (t Time) Month() Month { _, month, _ := t.Date(); return month }

// Day gives the day of the month of t in its location.
func (t Time) Day() int { _, _, day := t.Date(); return day }

// Date gives the year, the month, and the day of the month of t in its
// location.
func (t Time) Date() (int, Month, int) {
	_, _, wall := t.locWall()
	return dateOf(wall)
}

// Hour gives the hour of the day of t in its location. The range is 0 to 23.
func (t Time) Hour() int { hour, _, _ := t.Clock(); return hour }

// Minute gives the minute of the hour of t in its location.
func (t Time) Minute() int { _, min, _ := t.Clock(); return min }

// Second gives the second of the minute of t in its location.
func (t Time) Second() int { _, _, sec := t.Clock(); return sec }

// Clock gives the hour, the minute, and the second of t in its location.
func (t Time) Clock() (int, int, int) {
	_, _, wall := t.locWall()
	return clockOf(wall)
}

// Nanosecond gives the count of nanoseconds inside the second of t.
func (t Time) Nanosecond() int { return t.nanosecond() }

// Weekday gives the day of the week of t in its location.
func (t Time) Weekday() Weekday {
	_, _, wall := t.locWall()
	return weekdayOf(wall)
}

// YearDay gives the day of the year of t in its location. January 1 gives 1.
func (t Time) YearDay() int {
	_, _, wall := t.locWall()
	return yearDayOf(wall)
}

// shortDayNames holds the name of each day of the week of 3 letters. Parse
// reads it.
var shortDayNames = []string{
	"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat",
}

// daysOf gives the count of days since the start of the Unix epoch of a reading
// of a wall clock. The reading is a count of seconds.
func daysOf(wall int64) int64 { return floorDiv(wall, 86400) }

// dateOf gives the year, the month, and the day of a reading of a wall clock.
func dateOf(wall int64) (int, Month, int) { return civilFromDays(daysOf(wall)) }

// clockOf gives the hour, the minute, and the second of a reading of a wall
// clock.
func clockOf(wall int64) (int, int, int) {
	rest := wall - daysOf(wall)*86400
	return int(rest / 3600), int(rest / 60 % 60), int(rest % 60)
}

// weekdayOf gives the day of the week of a reading of a wall clock. January 1
// of 1970 was a Thursday.
func weekdayOf(wall int64) Weekday {
	return Weekday(floorMod(daysOf(wall)+4, 7))
}

// yearDayOf gives the day of the year of a reading of a wall clock. January 1
// gives 1.
func yearDayOf(wall int64) int {
	days := daysOf(wall)
	year, _, _ := civilFromDays(days)
	return int(days-daysFromCivil(int64(year), 1, 1)) + 1
}

// civilFromDays gives the year, the month, and the day of a count of days since
// the start of the Unix epoch. The algorithm of Howard Hinnant holds for each
// proleptic Gregorian date.
func civilFromDays(days int64) (int, Month, int) {
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
	return int(year), Month(month), int(day)
}

// daysFromCivil gives the count of days since the start of the Unix epoch of a
// year, a month, and a day. It is the other direction of civilFromDays, and a
// day outside the month stays correct, because the formula adds it.
func daysFromCivil(year int64, month, day int) int64 {
	shifted := int64(month - 3)
	if month <= 2 {
		year--
		shifted = int64(month + 9)
	}
	era := floorDiv(year, 400)
	yearOfEra := year - era*400
	dayOfYear := (153*shifted+2)/5 + int64(day) - 1
	dayOfEra := yearOfEra*365 + yearOfEra/4 - yearOfEra/100 + dayOfYear
	return era*146097 + dayOfEra - 719468
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

// floorMod gives the remainder of a division that rounds toward minus infinity.
func floorMod(a, b int64) int64 { return a - floorDiv(a, b)*b }

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

// Timer runs a function after a time, or it writes the time into its channel.
// The field C is nil for a timer of AfterFunc, exactly as in Go.
type Timer struct {
	C <-chan Time

	send chan Time
	// body holds the function of AfterFunc. Reset starts the timer again with
	// the same function, exactly as Go does.
	body   func()
	handle int
}

// Stop cancels a timer. It reports whether the timer was active or held
// an unread value.
func (t *Timer) Stop() bool {
	if t.send == nil && t.body == nil {
		panic("time: Stop called on uninitialized Timer")
	}
	active := dart.StopTimer(t.handle)
	return drainTimer(t.send) || active
}

// Reset stops the timer and starts it again with the new duration. It reports
// whether the timer was active or held an unread value.
func (t *Timer) Reset(d Duration) bool {
	if t.send == nil && t.body == nil {
		panic("time: Reset called on uninitialized Timer")
	}
	active := dart.ResetTimer(t.handle)
	active = drainTimer(t.send) || active
	if t.send == nil {
		t.handle = dart.StartTimer(int64(d), t.body)
		return active
	}
	t.handle = dart.StartTimer(int64(d), func() { sendTime(t.send) })
	return active
}

// AfterFunc runs f on a new goroutine after the duration.
func AfterFunc(d Duration, f func()) *Timer {
	return &Timer{body: f, handle: dart.StartTimer(int64(d), f)}
}

// NewTimer gives a timer that writes the time into its channel after the
// duration. The channel reports a capacity of 0.
func NewTimer(d Duration) *Timer {
	ch := make(chan Time, 1)
	dart.MarkTimerChannel(ch)
	t := &Timer{C: ch, send: ch}
	t.handle = dart.StartTimer(int64(d), func() { sendTime(ch) })
	return t
}

// After gives a channel that holds the time after the duration.
func After(d Duration) <-chan Time {
	return NewTimer(d).C
}

// sendTime writes the time into the channel. It drops the value when the
// channel is full, exactly as Go does for a timer.
func sendTime(ch chan Time) {
	dart.ValueTrySend(ch, Now())
}

func drainTimer(ch chan Time) bool {
	_, _, received := dart.ValueTryRecv(ch)
	return received
}

// A Ticker writes the time into its channel at each pass of a period.
type Ticker struct {
	C <-chan Time

	send    chan Time
	handle  int
	period  Duration
	stopped bool
}

// NewTicker gives a ticker of the period.
func NewTicker(d Duration) *Ticker {
	if d <= 0 {
		panic("non-positive interval for NewTicker")
	}
	ch := make(chan Time, 1)
	dart.MarkTimerChannel(ch)
	t := &Ticker{C: ch, send: ch, period: d}
	t.arm()
	return t
}

// Tick gives the channel of a new ticker. The ticker never stops, thus a caller
// that needs to stop it reads NewTicker.
func Tick(d Duration) <-chan Time {
	if d <= 0 {
		return nil
	}
	return NewTicker(d).C
}

// Stop ends the ticker. It closes no channel, exactly as Go does.
func (t *Ticker) Stop() {
	t.stopped = true
	dart.StopTimer(t.handle)
	drainTimer(t.send)
}

// Reset stops the ticker and starts it again with the new period.
func (t *Ticker) Reset(d Duration) {
	if d <= 0 {
		panic("non-positive interval for Reset")
	}
	dart.ResetTimer(t.handle)
	drainTimer(t.send)
	t.period = d
	t.stopped = false
	t.arm()
}

// arm starts the timer of the next pass. Each pass starts the timer again,
// because the interop package gives a timer of 1 run.
func (t *Ticker) arm() {
	t.handle = dart.StartTimer(int64(t.period), func() {
		if t.stopped {
			return
		}
		sendTime(t.send)
		t.arm()
	})
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
