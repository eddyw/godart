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

// A Time is a point in time. It holds a count of nanoseconds since the start of
// the Unix epoch and the location of its wall clock.
//
// Go holds a count of seconds since the year 1, thus the zero value of Go is
// that year. The zero value here is the start of the Unix epoch.
// docs/parity.md holds that gap.
type Time struct {
	nanos int64
	loc   *Location
}

// Now gives the time of the clock.
func Now() Time {
	return Time{nanos: dart.NowNanos(), loc: Local}
}

// Unix gives the time of a count of seconds and a count of nanoseconds since
// the start of the Unix epoch.
func Unix(sec, nsec int64) Time {
	return Time{nanos: sec*1000000000 + nsec, loc: Local}
}

// UnixNano gives the count of nanoseconds since the start of the Unix epoch.
func (t Time) UnixNano() int64 { return t.nanos }

// Unix gives the count of seconds since the start of the Unix epoch.
func (t Time) Unix() int64 { return t.nanos / 1000000000 }

// Sub gives the duration between t and u.
func (t Time) Sub(u Time) Duration { return Duration(t.nanos - u.nanos) }

// Add gives the time after the duration.
func (t Time) Add(d Duration) Time { return Time{nanos: t.nanos + int64(d), loc: t.loc} }

// Before reports whether t comes before u.
func (t Time) Before(u Time) bool { return t.nanos < u.nanos }

// After reports whether t comes after u.
func (t Time) After(u Time) bool { return t.nanos > u.nanos }

// Compare gives -1 when t comes before u, 0 when the 2 name the same instant,
// and 1 when t comes after u.
func (t Time) Compare(u Time) int {
	if t.nanos < u.nanos {
		return -1
	}
	if t.nanos > u.nanos {
		return 1
	}
	return 0
}

// IsZero reports whether t holds the zero time.
func (t Time) IsZero() bool { return t.nanos == 0 }

// String gives the text of the time, in the format of Go.
//
// Go also writes the reading of the monotonic clock, and a Time of the overlay
// holds no such reading. docs/parity.md holds that gap.
func (t Time) String() string {
	return t.Format("2006-01-02 15:04:05.999999999 -0700 MST")
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
	return Time{nanos: t.nanos, loc: loc}
}

// UTC gives t with the location UTC.
func (t Time) UTC() Time { return t.In(UTC) }

// Local gives t with the location of the system.
func (t Time) Local() Time { return t.In(Local) }

// Equal reports whether t and u name the same instant. The location of each
// one changes nothing.
func (t Time) Equal(u Time) bool { return t.nanos == u.nanos }

// locWall gives the name of the zone, the offset in seconds east of UTC, and
// the reading of the wall clock of the location in nanoseconds.
func (t Time) locWall() (string, int, int64) {
	offset, name := t.Location().lookup(t.nanos)
	return name, offset, t.nanos + int64(offset)*1000000000
}

// nanosecond gives the count of nanoseconds inside the second of t. A zone
// offset holds a whole count of seconds, thus the location changes nothing.
func (t Time) nanosecond() int {
	return int(floorMod(t.nanos, 1000000000))
}

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

	wall := daysFromCivil(int64(year), int(month), day)*86400000000000 +
		int64(hour)*3600000000000 + int64(min)*60000000000 +
		int64(sec)*1000000000 + int64(nsec)
	return Time{nanos: wall - int64(loc.offsetOf(wall))*1000000000, loc: loc}
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
// of a wall clock.
func daysOf(wall int64) int64 { return floorDiv(wall, 86400000000000) }

// dateOf gives the year, the month, and the day of a reading of a wall clock.
func dateOf(wall int64) (int, Month, int) { return civilFromDays(daysOf(wall)) }

// clockOf gives the hour, the minute, and the second of a reading of a wall
// clock.
func clockOf(wall int64) (int, int, int) {
	rest := wall - daysOf(wall)*86400000000000
	return int(rest / 3600000000000), int(rest / 60000000000 % 60), int(rest / 1000000000 % 60)
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

// Stop cancels a timer. It reports false when the timer ran already.
func (t *Timer) Stop() bool {
	if t.send == nil && t.body == nil {
		panic("time: Stop called on uninitialized Timer")
	}
	return dart.StopTimer(t.handle)
}

// Reset stops the timer and starts it again with the new duration. It reports
// whether the timer was active.
func (t *Timer) Reset(d Duration) bool {
	if t.send == nil && t.body == nil {
		panic("time: Reset called on uninitialized Timer")
	}
	active := dart.ResetTimer(t.handle)
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
// duration. The channel holds 1 value, thus the write never blocks.
func NewTimer(d Duration) *Timer {
	ch := make(chan Time, 1)
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
	select {
	case ch <- Now():
	default:
	}
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
}

// Reset stops the ticker and starts it again with the new period.
func (t *Ticker) Reset(d Duration) {
	if d <= 0 {
		panic("non-positive interval for Reset")
	}
	dart.ResetTimer(t.handle)
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
