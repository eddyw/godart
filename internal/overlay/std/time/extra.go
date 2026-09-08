//go:build dart

package time

// Hours gives the count of hours of the duration, as a float.
func (d Duration) Hours() float64 {
	hour := d / Hour
	rest := d % Hour
	return float64(hour) + float64(rest)/(60*60*1e9)
}

// Minutes gives the count of minutes of the duration, as a float.
func (d Duration) Minutes() float64 {
	minute := d / Minute
	rest := d % Minute
	return float64(minute) + float64(rest)/(60*1e9)
}

// Abs gives the duration with no sign. The lowest value of an int64 holds no
// positive form, thus it gives the highest value.
func (d Duration) Abs() Duration {
	switch {
	case d >= 0:
		return d
	case d == -1<<63:
		return 1<<63 - 1
	}
	return -d
}

// UnixMilli gives the count of milliseconds since the start of the Unix epoch.
func (t Time) UnixMilli() int64 {
	return t.unixSec()*1000 + int64(t.nsec)/1000000
}

// UnixMicro gives the count of microseconds since the start of the Unix epoch.
func (t Time) UnixMicro() int64 {
	return t.unixSec()*1000000 + int64(t.nsec)/1000
}

// UnixMilli gives the Time of a count of milliseconds since the start of the
// Unix epoch.
func UnixMilli(msec int64) Time {
	return Unix(floorDiv(msec, 1000), floorMod(msec, 1000)*1000000)
}

// UnixMicro gives the Time of a count of microseconds since the start of the
// Unix epoch.
func UnixMicro(usec int64) Time {
	return Unix(floorDiv(usec, 1000000), floorMod(usec, 1000000)*1000)
}

// AddDate gives the time after the count of years, months, and days. A value
// outside its range moves into the next unit, exactly as in Go.
func (t Time) AddDate(years int, months int, days int) Time {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	return Date(year+years, month+Month(months), day+days,
		hour, min, sec, t.Nanosecond(), t.Location())
}

// Zone gives the name of the zone of the wall clock of t and its offset in
// seconds east of UTC.
func (t Time) Zone() (string, int) {
	offset, name := t.Location().lookup(t.unixSec())
	return name, offset
}

// ZoneBounds gives the start and the end of the zone of t. A zone with no start
// gives the zero Time, and a zone with no end gives the zero Time.
func (t Time) ZoneBounds() (Time, Time) {
	loc := t.Location()
	if len(loc.zone) == 0 {
		// A location of a fixed offset and the local zone each hold 1 zone for
		// all time, thus the 2 bounds stay empty.
		return Time{}, Time{}
	}
	_, _, start, end, _ := loc.lookupSec(t.unixSec())
	var from, to Time
	if start != alpha {
		from = Time{sec: start + unixToInternal, loc: t.loc}
	}
	if end != omega {
		to = Time{sec: end + unixToInternal, loc: t.loc}
	}
	return from, to
}

// IsDST reports whether the zone of t holds daylight saving time.
func (t Time) IsDST() bool {
	loc := t.Location()
	if len(loc.zone) == 0 {
		return false
	}
	_, _, _, _, isDST := loc.lookupSec(t.unixSec())
	return isDST
}

// ISOWeek gives the year and the week of t in the ISO 8601 calendar. Week 1
// holds the first Thursday of the year, and a week starts on Monday.
func (t Time) ISOWeek() (int, int) {
	// The Thursday of the week of t names the year of the week, thus the count
	// of the days of that year gives the number of the week.
	day := int(t.Weekday()+6) % 7
	thursday := t.AddDate(0, 0, 3-day)
	year, _, _ := thursday.Date()
	return year, (thursday.YearDay()-1)/7 + 1
}

// AppendFormat writes the text of the layout after b.
func (t Time) AppendFormat(b []byte, layout string) []byte {
	return t.appendFormat(b, layout)
}

// GoString gives the text of a Go source expression that builds t.
func (t Time) GoString() string {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	out := "time.Date(" + itoa(int64(year)) + ", time." + month.String() + ", " +
		itoa(int64(day)) + ", " + itoa(int64(hour)) + ", " + itoa(int64(min)) + ", " +
		itoa(int64(sec)) + ", " + itoa(int64(t.Nanosecond())) + ", "
	loc := t.Location()
	switch {
	case loc == UTC || loc == nil:
		return out + "time.UTC)"
	case loc == Local:
		return out + "time.Local)"
	}
	return out + "time.Location(" + quote(loc.name) + "))"
}

// GobEncode gives the binary form of t.
func (t Time) GobEncode() ([]byte, error) { return t.MarshalBinary() }

// GobDecode reads the binary form of a time into t.
func (t *Time) GobDecode(data []byte) error { return t.UnmarshalBinary(data) }
