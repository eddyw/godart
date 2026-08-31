//go:build dart

package time

import (
	"errors"
	"syscall/dart"
)

// A Location names a time zone. UTC and Local are the 2 built locations.
// LoadLocation builds a location from the zone database of the system or from
// the embedded database of time/tzdata.
type Location struct {
	name string
	// local marks the zone of the system. Such a zone reads its offset from
	// the Dart runtime at each instant, thus a change of the offset over the
	// year reaches the output.
	local bool
	// offset holds the seconds east of UTC of a zone of a fixed offset. Parse
	// builds such a zone when it reads an offset that the local zone does not
	// hold.
	offset int

	// The fields below hold a zone database entry. LoadLocationFromTZData
	// fills them, and lookupSec reads them. The comments of zoneinfo.go of Go
	// describe each one.
	zone       []zone
	tx         []zoneTrans
	extend     string
	cacheStart int64
	cacheEnd   int64
	cacheZone  *zone
}

// A zone holds a single name and offset of a location, such as CET.
type zone struct {
	name   string
	offset int
	isDST  bool
}

// A zoneTrans holds a single transition of a location.
type zoneTrans struct {
	when         int64
	index        uint8
	isstd, isutc bool
}

// alpha and omega are the start and the end of time for a transition.
const (
	alpha = -1 << 63
	omega = 1<<63 - 1
)

// UTC holds the coordinated universal time.
var UTC = &Location{name: "UTC"}

// Local holds the time zone of the system.
var Local = &Location{name: "Local", local: true}

// FixedZone gives a Location that always reads the offset in seconds east of
// UTC.
func FixedZone(name string, offset int) *Location {
	return &Location{name: name, offset: offset}
}

// String gives the name of the location.
func (l *Location) String() string {
	if l == nil {
		return "UTC"
	}
	return l.name
}

// lookup gives the offset in seconds east of UTC and the name of the zone at an
// instant of the Unix epoch.
func (l *Location) lookup(nanos int64) (int, string) {
	if l == nil {
		return 0, "UTC"
	}
	if l.local {
		return dart.ZoneOffset(nanos)
	}
	if len(l.zone) == 0 {
		return l.offset, l.name
	}
	name, offset, _, _, _ := l.lookupSec(floorDiv(nanos, 1000000000))
	return offset, name
}

// lookupSec is the lookup of zoneinfo.go of Go. The argument is an instant in
// seconds since the start of the Unix epoch, and the results are the name of
// the zone, the offset, the start and the end of the zone, and the DST flag.
func (l *Location) lookupSec(sec int64) (name string, offset int, start, end int64, isDST bool) {
	if zone := l.cacheZone; zone != nil && l.cacheStart <= sec && sec < l.cacheEnd {
		return zone.name, zone.offset, l.cacheStart, l.cacheEnd, zone.isDST
	}

	if len(l.tx) == 0 || sec < l.tx[0].when {
		zone := &l.zone[l.lookupFirstZone()]
		end = omega
		if len(l.tx) > 0 {
			end = l.tx[0].when
		}
		return zone.name, zone.offset, alpha, end, zone.isDST
	}

	// A binary search finds the last transition at or before sec.
	tx := l.tx
	end = omega
	lo := 0
	hi := len(tx)
	for hi-lo > 1 {
		m := int(uint(lo+hi) >> 1)
		lim := tx[m].when
		if sec < lim {
			end = lim
			hi = m
		} else {
			lo = m
		}
	}
	zone := &l.zone[tx[lo].index]
	name = zone.name
	offset = zone.offset
	start = tx[lo].when
	isDST = zone.isDST

	// The extend rule covers an instant after the last transition.
	if lo == len(tx)-1 && l.extend != "" {
		if ename, eoffset, estart, eend, eisDST, ok := tzset(l.extend, start, sec); ok {
			return ename, eoffset, estart, eend, eisDST
		}
	}

	return
}

// lookupFirstZone gives the index of the zone in use before the first
// transition. The rule comes from localtime.c of the IANA code, through
// zoneinfo.go of Go.
func (l *Location) lookupFirstZone() int {
	if !l.firstZoneUsed() {
		return 0
	}

	if len(l.tx) > 0 && l.zone[l.tx[0].index].isDST {
		for zi := int(l.tx[0].index) - 1; zi >= 0; zi-- {
			if !l.zone[zi].isDST {
				return zi
			}
		}
	}

	for zi := range l.zone {
		if !l.zone[zi].isDST {
			return zi
		}
	}

	return 0
}

// firstZoneUsed reports whether a transition reads the first zone.
func (l *Location) firstZoneUsed() bool {
	for _, tx := range l.tx {
		if tx.index == 0 {
			return true
		}
	}
	return false
}

// lookupName gives the offset of a zone of the location that carries the name
// at an instant. It reports whether the location holds such a zone.
func (l *Location) lookupName(name string, nanos int64) (int, bool) {
	if l == nil {
		return 0, false
	}
	if l.local {
		offset, zone := dart.ZoneOffset(nanos)
		if zone != name {
			return 0, false
		}
		return offset, true
	}
	unix := floorDiv(nanos, 1000000000)

	// The first pass takes a zone of the name that is in effect at the
	// instant, because 2 zones of a location can share a name.
	for i := range l.zone {
		zone := &l.zone[i]
		if zone.name == name {
			nam, offset, _, _, _ := l.lookupSec(unix - int64(zone.offset))
			if nam == zone.name {
				return offset, true
			}
		}
	}

	for i := range l.zone {
		zone := &l.zone[i]
		if zone.name == name {
			return zone.offset, true
		}
	}

	return 0, false
}

// offsetOf gives the offset of the location at a reading of its wall clock, in
// nanoseconds as if the reading were UTC.
//
// The Dart runtime names the offset of an instant, and a wall clock reading is
// not such an instant. The first lookup reads the wall clock as if it were UTC,
// and the second one reads the instant that the first offset gives. A reading
// inside 1 offset of a transition of the zone can take the offset of the other
// side. docs/parity.md holds that gap. A loaded zone holds the transitions,
// thus it takes the exact rule of Date of Go.
func (l *Location) offsetOf(wall int64) int {
	if l == nil {
		return 0
	}
	if l.local {
		guess, _ := dart.ZoneOffset(wall)
		offset, _ := dart.ZoneOffset(wall - int64(guess)*1000000000)
		return offset
	}
	if len(l.zone) == 0 {
		return l.offset
	}
	sec := floorDiv(wall, 1000000000)
	_, offset, start, end, _ := l.lookupSec(sec)
	if offset != 0 {
		switch utc := sec - int64(offset); {
		case utc < start:
			_, offset, _, _, _ = l.lookupSec(start - 1)
		case utc >= end:
			_, offset, _, _, _ = l.lookupSec(end)
		}
	}
	return offset
}

// LoadLocation gives the location of a name of the IANA database, such as
// America/New_York.
//
// The load reads the directory or the zip file of the variable ZONEINFO first,
// then each zone directory of the system, and then the database that
// time/tzdata embeds, when the program imports that package.
func LoadLocation(name string) (*Location, error) {
	if name == "" || name == "UTC" {
		return UTC, nil
	}
	if name == "Local" {
		return Local, nil
	}
	if containsDotDot(name) || name[0] == '/' || name[0] == '\\' {
		// No name of the IANA database holds a dot pair or starts at the root.
		return nil, errLocation
	}
	var firstErr error
	if env, found := dart.Getenv("ZONEINFO"); found && env != "" {
		if zoneData, err := loadTzinfo(name, env); err == nil {
			if z, err := LoadLocationFromTZData(name, zoneData); err == nil {
				return z, nil
			}
			firstErr = err
		} else if !isNotFound(err) {
			firstErr = err
		}
	}
	if z, err := loadLocation(name, platformZoneSources); err == nil {
		return z, nil
	} else if firstErr == nil {
		firstErr = err
	}
	return nil, firstErr
}

var errLocation = errors.New("time: invalid location name")

// platformZoneSources holds the zone directories of a Unix system. Windows
// holds no such directory, thus a program there needs time/tzdata.
var platformZoneSources = []string{
	"/usr/share/zoneinfo/",
	"/usr/share/lib/zoneinfo/",
	"/usr/lib/locale/TZ/",
}

func containsDotDot(s string) bool {
	if len(s) < 2 {
		return false
	}
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '.' && s[i+1] == '.' {
			return true
		}
	}
	return false
}

// tzset reads a rule string of the TZ form, such as PST8PDT,M3.2.0,M11.1.0.
// The code is the tzset chain of zoneinfo.go of Go. The 2 instants are the
// last transition and the instant of the lookup, in seconds since the start of
// the Unix epoch.
func tzset(s string, lastTxSec, sec int64) (name string, offset int, start, end int64, isDST, ok bool) {
	var (
		stdName, dstName     string
		stdOffset, dstOffset int
	)

	stdName, s, ok = tzsetName(s)
	if ok {
		stdOffset, s, ok = tzsetOffset(s)
	}
	if !ok {
		return "", 0, 0, 0, false, false
	}

	// The numbers of the rule string add to the local clock, and an offset of
	// a location adds to UTC, thus the sign turns.
	stdOffset = -stdOffset

	if len(s) == 0 || s[0] == ',' {
		return stdName, stdOffset, lastTxSec, omega, false, true
	}

	dstName, s, ok = tzsetName(s)
	if ok {
		if len(s) == 0 || s[0] == ',' {
			dstOffset = stdOffset + secondsPerHour
		} else {
			dstOffset, s, ok = tzsetOffset(s)
			dstOffset = -dstOffset
		}
	}
	if !ok {
		return "", 0, 0, 0, false, false
	}

	if len(s) == 0 {
		// The default DST rule of the IANA code.
		s = ",M3.2.0,M11.1.0"
	}
	if s[0] != ',' && s[0] != ';' {
		return "", 0, 0, 0, false, false
	}
	s = s[1:]

	var startRule, endRule rule
	startRule, s, ok = tzsetRule(s)
	if !ok || len(s) == 0 || s[0] != ',' {
		return "", 0, 0, 0, false, false
	}
	s = s[1:]
	endRule, s, ok = tzsetRule(s)
	if !ok || len(s) > 0 {
		return "", 0, 0, 0, false, false
	}

	// The year of sec, and the seconds since the start of that year.
	days := floorDiv(sec, secondsPerDay)
	y, _, _ := civilFromDays(days)
	year := int64(y)
	ystart := daysFromCivil(year, 1, 1) * secondsPerDay
	ysec := sec - ystart

	startSec := int64(tzruleTime(int(year), startRule, stdOffset))
	endSec := int64(tzruleTime(int(year), endRule, dstOffset))
	dstIsDST, stdIsDST := true, false
	// The southern hemisphere holds the DST period over the turn of the year,
	// thus the 2 rules swap.
	if endSec < startSec {
		startSec, endSec = endSec, startSec
		stdName, dstName = dstName, stdName
		stdOffset, dstOffset = dstOffset, stdOffset
		stdIsDST, dstIsDST = dstIsDST, stdIsDST
	}

	if ysec < startSec {
		return stdName, stdOffset, ystart, startSec + ystart, stdIsDST, true
	} else if ysec >= endSec {
		return stdName, stdOffset, endSec + ystart, ystart + 365*secondsPerDay, stdIsDST, true
	} else {
		return dstName, dstOffset, startSec + ystart, endSec + ystart, dstIsDST, true
	}
}

// tzsetName reads the name at the start of a rule string.
func tzsetName(s string) (string, string, bool) {
	if len(s) == 0 {
		return "", "", false
	}
	if s[0] != '<' {
		for i, r := range s {
			switch r {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', ',', '-', '+':
				if i < 3 {
					return "", "", false
				}
				return s[:i], s[i:], true
			}
		}
		if len(s) < 3 {
			return "", "", false
		}
		return s, "", true
	}
	for i, r := range s {
		if r == '>' {
			return s[1:i], s[i+1:], true
		}
	}
	return "", "", false
}

// tzsetOffset reads the offset at the start of a rule string, in seconds.
func tzsetOffset(s string) (offset int, rest string, ok bool) {
	if len(s) == 0 {
		return 0, "", false
	}
	neg := false
	if s[0] == '+' {
		s = s[1:]
	} else if s[0] == '-' {
		s = s[1:]
		neg = true
	}

	// The IANA code permits a value up to 24 * 7 here, and POSIX does not.
	var hours int
	hours, s, ok = tzsetNum(s, 0, 24*7)
	if !ok {
		return 0, "", false
	}
	off := hours * secondsPerHour
	if len(s) == 0 || s[0] != ':' {
		if neg {
			off = -off
		}
		return off, s, true
	}

	var mins int
	mins, s, ok = tzsetNum(s[1:], 0, 59)
	if !ok {
		return 0, "", false
	}
	off += mins * secondsPerMinute
	if len(s) == 0 || s[0] != ':' {
		if neg {
			off = -off
		}
		return off, s, true
	}

	var secs int
	secs, s, ok = tzsetNum(s[1:], 0, 59)
	if !ok {
		return 0, "", false
	}
	off += secs

	if neg {
		off = -off
	}
	return off, s, true
}

// ruleKind names the 3 forms of a transition rule.
type ruleKind int

const (
	ruleJulian ruleKind = iota
	ruleDOY
	ruleMonthWeekDay
)

// rule holds a transition rule of a rule string.
type rule struct {
	kind ruleKind
	day  int
	week int
	mon  int
	time int
}

// tzsetRule reads a transition rule at the start of a rule string.
func tzsetRule(s string) (rule, string, bool) {
	var r rule
	if len(s) == 0 {
		return rule{}, "", false
	}
	ok := false
	if s[0] == 'J' {
		var jday int
		jday, s, ok = tzsetNum(s[1:], 1, 365)
		if !ok {
			return rule{}, "", false
		}
		r.kind = ruleJulian
		r.day = jday
	} else if s[0] == 'M' {
		var mon int
		mon, s, ok = tzsetNum(s[1:], 1, 12)
		if !ok || len(s) == 0 || s[0] != '.' {
			return rule{}, "", false
		}
		var week int
		week, s, ok = tzsetNum(s[1:], 1, 5)
		if !ok || len(s) == 0 || s[0] != '.' {
			return rule{}, "", false
		}
		var day int
		day, s, ok = tzsetNum(s[1:], 0, 6)
		if !ok {
			return rule{}, "", false
		}
		r.kind = ruleMonthWeekDay
		r.day = day
		r.week = week
		r.mon = mon
	} else {
		var day int
		day, s, ok = tzsetNum(s, 0, 365)
		if !ok {
			return rule{}, "", false
		}
		r.kind = ruleDOY
		r.day = day
	}

	if len(s) == 0 || s[0] != '/' {
		r.time = 2 * secondsPerHour
		return r, s, true
	}

	offset, s, ok := tzsetOffset(s[1:])
	if !ok {
		return rule{}, "", false
	}
	r.time = offset

	return r, s, true
}

// tzsetNum reads a number at the start of a rule string. The number must stay
// between min and max.
func tzsetNum(s string, min, max int) (num int, rest string, ok bool) {
	if len(s) == 0 {
		return 0, "", false
	}
	num = 0
	for i, r := range s {
		if r < '0' || r > '9' {
			if i == 0 || num < min {
				return 0, "", false
			}
			return num, s[i:], true
		}
		num *= 10
		num += int(r) - '0'
		if num > max {
			return 0, "", false
		}
	}
	if num < min {
		return 0, "", false
	}
	return num, "", true
}

// tzruleTime gives the seconds since the start of the year at which a rule
// takes effect.
func tzruleTime(year int, r rule, off int) int {
	var s int
	switch r.kind {
	case ruleJulian:
		s = (r.day - 1) * secondsPerDay
		if isLeap(year) && r.day >= 60 {
			s += secondsPerDay
		}
	case ruleDOY:
		s = r.day * secondsPerDay
	case ruleMonthWeekDay:
		// The congruence of Zeller gives the weekday of the first day of the
		// month.
		m1 := (r.mon+9)%12 + 1
		yy0 := year
		if r.mon <= 2 {
			yy0--
		}
		yy1 := yy0 / 100
		yy2 := yy0 % 100
		dow := ((26*m1-2)/10 + 1 + yy2 + yy2/4 + yy1/4 - 2*yy1) % 7
		if dow < 0 {
			dow += 7
		}
		d := r.day - dow
		if d < 0 {
			d += 7
		}
		for i := 1; i < r.week; i++ {
			if d+7 >= daysIn(Month(r.mon), year) {
				break
			}
			d += 7
		}
		d += daysBefore(Month(r.mon))
		if isLeap(year) && r.mon > 2 {
			d++
		}
		s = d * secondsPerDay
	}

	return s + r.time - off
}

// isLeap, daysBefore, and daysIn live in parse.go of the overlay.
const (
	secondsPerMinute = 60
	secondsPerHour   = 60 * secondsPerMinute
	secondsPerDay    = 24 * secondsPerHour
)
